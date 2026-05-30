package models

type AnalysisResults struct {
	UnusedFunctions    []*Entity
	UnusedSignals      []*Entity
	UnusedConstants    []*Entity
	UnusedEnums        []*Entity
	UnusedEnumValues   []*Entity
	UnusedClassNames   []*Entity
	UnusedFiles        []string
	PrintStatements    []CodeLocation
	Comments           []CodeLocation
	PassStatements     []CodeLocation
	OrphanedUIDs       []string
	IndentationIssues  []CodeLocation
	HasMethodCalls     []CodeLocation
	VariantUsages      []CodeLocation
	MissingReturnTypes []CodeLocation

	Warnings Warnings
}

type Warnings struct {
	SingleUseFunctions []*Entity
	EmptyFunctions     []*Entity
}

type CodeLocation struct {
	File   string
	Line   int
	Column int
	Text   string
}

func NewAnalysisResults() *AnalysisResults {
	return &AnalysisResults{
		UnusedFunctions:    make([]*Entity, 0),
		UnusedSignals:      make([]*Entity, 0),
		UnusedConstants:    make([]*Entity, 0),
		UnusedEnums:        make([]*Entity, 0),
		UnusedEnumValues:   make([]*Entity, 0),
		UnusedClassNames:   make([]*Entity, 0),
		UnusedFiles:        make([]string, 0),
		PrintStatements:    make([]CodeLocation, 0),
		Comments:           make([]CodeLocation, 0),
		PassStatements:     make([]CodeLocation, 0),
		OrphanedUIDs:       make([]string, 0),
		IndentationIssues:  make([]CodeLocation, 0),
		HasMethodCalls:     make([]CodeLocation, 0),
		VariantUsages:      make([]CodeLocation, 0),
		MissingReturnTypes: make([]CodeLocation, 0),
		Warnings: Warnings{
			SingleUseFunctions: make([]*Entity, 0),
			EmptyFunctions:     make([]*Entity, 0),
		},
	}
}

func (r *AnalysisResults) HasErrors() bool {
	return len(r.UnusedFunctions) > 0 ||
		len(r.UnusedSignals) > 0 ||
		len(r.UnusedConstants) > 0 ||
		len(r.UnusedEnums) > 0 ||
		len(r.UnusedEnumValues) > 0 ||
		len(r.UnusedClassNames) > 0 ||
		len(r.UnusedFiles) > 0 ||
		len(r.PrintStatements) > 0 ||
		len(r.Comments) > 0 ||
		len(r.PassStatements) > 0 ||
		len(r.OrphanedUIDs) > 0 ||
		len(r.IndentationIssues) > 0 ||
		len(r.HasMethodCalls) > 0 ||
		len(r.VariantUsages) > 0 ||
		len(r.MissingReturnTypes) > 0
}

func (r *AnalysisResults) HasWarnings() bool {
	return len(r.Warnings.SingleUseFunctions) > 0 ||
		len(r.Warnings.EmptyFunctions) > 0
}

func (r *AnalysisResults) ErrorCount() int {
	count := 0
	count += len(r.UnusedFunctions)
	count += len(r.UnusedSignals)
	count += len(r.UnusedConstants)
	count += len(r.UnusedEnums)
	count += len(r.UnusedEnumValues)
	count += len(r.UnusedClassNames)
	count += len(r.UnusedFiles)
	count += len(r.PrintStatements)
	count += len(r.Comments)
	count += len(r.PassStatements)
	count += len(r.OrphanedUIDs)
	count += len(r.IndentationIssues)
	count += len(r.HasMethodCalls)
	count += len(r.VariantUsages)
	count += len(r.MissingReturnTypes)
	return count
}

func (r *AnalysisResults) WarningCount() int {
	count := 0
	count += len(r.Warnings.SingleUseFunctions)
	count += len(r.Warnings.EmptyFunctions)
	return count
}
