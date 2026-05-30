package optimization

import (
	"runtime"
	"sync"
)

type WorkerPool struct {
	workers   int
	workQueue chan WorkItem
	wg        sync.WaitGroup
}

type WorkItem struct {
	ID   int
	Data interface{}
	Func func(interface{}) interface{}
}

type WorkResult struct {
	ID     int
	Result interface{}
	Error  error
}

func NewWorkerPool(workers int) *WorkerPool {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	
	return &WorkerPool{
		workers:   workers,
		workQueue: make(chan WorkItem, workers*2),
	}
}

func (p *WorkerPool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

func (p *WorkerPool) Stop() {
	close(p.workQueue)
	p.wg.Wait()
}

func (p *WorkerPool) Submit(item WorkItem) {
	p.workQueue <- item
}

func (p *WorkerPool) worker() {
	defer p.wg.Done()
	
	for item := range p.workQueue {
		item.Func(item.Data)
	}
}

func (p *WorkerPool) ProcessBatch(items []WorkItem, results chan<- WorkResult) {
	var wg sync.WaitGroup
	
	for _, item := range items {
		wg.Add(1)
		go func(wi WorkItem) {
			defer wg.Done()
			
			result := wi.Func(wi.Data)
			results <- WorkResult{
				ID:     wi.ID,
				Result: result,
				Error:  nil,
			}
		}(item)
	}
	
	go func() {
		wg.Wait()
		close(results)
	}()
}

type ParallelProcessor struct {
	maxWorkers int
}

func NewParallelProcessor(maxWorkers int) *ParallelProcessor {
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	
	return &ParallelProcessor{
		maxWorkers: maxWorkers,
	}
}

func (p *ParallelProcessor) Process(items []interface{}, fn func(interface{}) interface{}) []interface{} {
	if len(items) == 0 {
		return []interface{}{}
	}
	
	if len(items) == 1 {
		return []interface{}{fn(items[0])}
	}
	
	numWorkers := p.maxWorkers
	if len(items) < numWorkers {
		numWorkers = len(items)
	}
	
	resultsChan := make(chan WorkResult, len(items))
	results := make([]interface{}, len(items))
	
	workItems := make([]WorkItem, len(items))
	for i, item := range items {
		workItems[i] = WorkItem{
			ID:   i,
			Data: item,
			Func: fn,
		}
	}
	
	pool := NewWorkerPool(numWorkers)
	pool.Start()
	pool.ProcessBatch(workItems, resultsChan)
	pool.Stop()
	
	for result := range resultsChan {
		results[result.ID] = result.Result
	}
	
	return results
}

func (p *ParallelProcessor) Map(items []string, fn func(string) string) []string {
	if len(items) == 0 {
		return []string{}
	}
	
	interfaces := make([]interface{}, len(items))
	for i, item := range items {
		interfaces[i] = item
	}
	
	wrappedFn := func(item interface{}) interface{} {
		return fn(item.(string))
	}
	
	results := p.Process(interfaces, wrappedFn)
	
	strings := make([]string, len(results))
	for i, result := range results {
		if result != nil {
			strings[i] = result.(string)
		}
	}
	
	return strings
}

func (p *ParallelProcessor) Filter(items []string, fn func(string) bool) []string {
	if len(items) == 0 {
		return []string{}
	}
	
	type filterResult struct {
		item string
		keep bool
	}
	
	interfaces := make([]interface{}, len(items))
	for i, item := range items {
		interfaces[i] = item
	}
	
	wrappedFn := func(item interface{}) interface{} {
		str := item.(string)
		return filterResult{
			item: str,
			keep: fn(str),
		}
	}
	
	results := p.Process(interfaces, wrappedFn)
	
	var filtered []string
	for _, result := range results {
		if result != nil {
			fr := result.(filterResult)
			if fr.keep {
				filtered = append(filtered, fr.item)
			}
		}
	}
	
	return filtered
}