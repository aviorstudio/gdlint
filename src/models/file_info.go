package models

import (
	"path/filepath"
	"strings"
)

type FileInfo struct {
	Path         string
	RelativePath string
	Content      string
	Lines        []string
	IsAutoload   bool
	ScriptClass  string
	Entities     []*Entity
	Usages       map[string][]Usage
}

type Usage struct {
	Entity    string
	File      string
	Line      int
	Column    int
	UsageType UsageType
}

type UsageType string

const (
	UsageCall      UsageType = "call"
	UsageReference UsageType = "reference"
	UsageEmit      UsageType = "emit"
	UsageConnect   UsageType = "connect"
	UsageString    UsageType = "string"
)

func NewFileInfo(path string, rootDir string) *FileInfo {
	relPath, _ := filepath.Rel(rootDir, path)

	return &FileInfo{
		Path:         path,
		RelativePath: relPath,
		Entities:     make([]*Entity, 0),
		Usages:       make(map[string][]Usage),
	}
}

func (f *FileInfo) AddEntity(entity *Entity) {
	f.Entities = append(f.Entities, entity)
}

func (f *FileInfo) AddUsage(entityName string, usage Usage) {
	f.Usages[entityName] = append(f.Usages[entityName], usage)
}

func (f *FileInfo) GetEntitiesByType(entityType EntityType) []*Entity {
	var results []*Entity
	for _, entity := range f.Entities {
		if entity.Type == entityType {
			results = append(results, entity)
		}
	}
	return results
}

func (f *FileInfo) HasContent() bool {
	return len(strings.TrimSpace(f.Content)) > 0
}
