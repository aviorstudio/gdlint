package models

import "fmt"

type EntityType string

const (
	EntityFunction EntityType = "function"
	EntitySignal   EntityType = "signal"
	EntityConstant EntityType = "constant"
	EntityEnum     EntityType = "enum"
	EntityEnumValue EntityType = "enum_value"
	EntityClassName EntityType = "class_name"
)

type Entity struct {
	Type       EntityType
	Name       string
	File       string
	Line       int
	Column     int
	EndLine    int
	IndentLevel int
	Parent     string
	Value      string
	IsStatic   bool
	IsRPC      bool
	IsExported bool
}

func (e *Entity) String() string {
	return fmt.Sprintf("%s.%s (%s:%d)", e.Parent, e.Name, e.File, e.Line)
}

func (e *Entity) FullName() string {
	if e.Parent != "" {
		return fmt.Sprintf("%s.%s", e.Parent, e.Name)
	}
	return e.Name
}

func (e *Entity) IsProtected() bool {
	// Class names are always protected
	if e.Type == EntityClassName {
		return true
	}
	
	if e.Type != EntityFunction {
		return false
	}
	
	protectedPrefixes := []string{
		"_ready", "_init", "_enter_tree", "_exit_tree",
		"_process", "_physics_process", "_input", "_unhandled_input",
		"_draw", "_gui_input", "_notification",
	}
	
	for _, prefix := range protectedPrefixes {
		if e.Name == prefix {
			return true
		}
	}
	
	if len(e.Name) >= 4 && e.Name[:4] == "_on_" {
		return true
	}
	
	// Protect getter/setter functions (matches Python linter behavior)
	if len(e.Name) >= 4 && (e.Name[:4] == "get_" || e.Name[:4] == "set_") {
		return true
	}
	
	if e.IsRPC {
		return true
	}
	
	return false
}

type EntityMap map[string][]*Entity

func (m EntityMap) Add(entity *Entity) {
	key := entity.FullName()
	m[key] = append(m[key], entity)
}

func (m EntityMap) Get(name string) []*Entity {
	return m[name]
}

func (m EntityMap) GetByType(entityType EntityType) []*Entity {
	var results []*Entity
	for _, entities := range m {
		for _, entity := range entities {
			if entity.Type == entityType {
				results = append(results, entity)
			}
		}
	}
	return results
}

func (m EntityMap) Count() int {
	count := 0
	for _, entities := range m {
		count += len(entities)
	}
	return count
}