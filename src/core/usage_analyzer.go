package core

import (
	"fmt"
	"strings"

	"github.com/aviorstudio/gdlint/src/models"
	"github.com/aviorstudio/gdlint/src/patterns"
)

type UsageAnalyzer struct {
	files    []*models.FileInfo
	entities models.EntityMap
	usages   map[string][]models.Usage
}

func NewUsageAnalyzer(files []*models.FileInfo) *UsageAnalyzer {
	entities := make(models.EntityMap)
	for _, file := range files {
		for _, entity := range file.Entities {
			entities.Add(entity)
		}
	}

	return &UsageAnalyzer{
		files:    files,
		entities: entities,
		usages:   make(map[string][]models.Usage),
	}
}

func (a *UsageAnalyzer) AnalyzeUsages() {
	for _, file := range a.files {
		a.analyzeFile(file)
	}
}

func (a *UsageAnalyzer) analyzeFile(file *models.FileInfo) {
	for i, line := range file.Lines {
		lineNum := i + 1
		cleanLine := patterns.RemoveComments(line)

		a.detectFunctionCalls(file, cleanLine, lineNum)
		a.detectSignalUsage(file, cleanLine, lineNum)
		a.detectConnectedCallbacks(file, cleanLine, lineNum)
		a.detectCallableReferences(file, cleanLine, lineNum)
		a.detectConstantUsage(file, cleanLine, lineNum)
		a.detectEnumUsage(file, cleanLine, lineNum)
		a.detectStringReferences(file, cleanLine, lineNum)
	}
}

func (a *UsageAnalyzer) detectFunctionCalls(file *models.FileInfo, line string, lineNum int) {
	matches := patterns.FunctionCallPattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) > 1 {
			funcName := match[1]

			if patterns.IsProtectedFunction(funcName) {
				continue
			}

			usage := models.Usage{
				Entity:    funcName,
				File:      file.RelativePath,
				Line:      lineNum,
				UsageType: models.UsageCall,
			}

			a.addUsage(funcName, usage)
			file.AddUsage(funcName, usage)
		}
	}
}

func (a *UsageAnalyzer) detectSignalUsage(file *models.FileInfo, line string, lineNum int) {
	emitMatches := patterns.SignalEmitPattern.FindAllStringSubmatch(line, -1)
	for _, match := range emitMatches {
		if len(match) > 1 {
			signalName := match[1]
			usage := models.Usage{
				Entity:    signalName,
				File:      file.RelativePath,
				Line:      lineNum,
				UsageType: models.UsageEmit,
			}
			a.addUsage(signalName, usage)
			file.AddUsage(signalName, usage)
		}
	}

	connectMatches := patterns.SignalConnectPattern.FindAllStringSubmatch(line, -1)
	for _, match := range connectMatches {
		if len(match) > 1 {
			signalName := match[1]
			usage := models.Usage{
				Entity:    signalName,
				File:      file.RelativePath,
				Line:      lineNum,
				UsageType: models.UsageConnect,
			}
			a.addUsage(signalName, usage)
			file.AddUsage(signalName, usage)
		}
	}
}

func (a *UsageAnalyzer) detectConnectedCallbacks(file *models.FileInfo, line string, lineNum int) {
	matches := patterns.ConnectCallbackPattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) > 1 {
			callback := match[1]
			if patterns.IsProtectedFunction(callback) {
				continue
			}

			usage := models.Usage{
				Entity:    callback,
				File:      file.RelativePath,
				Line:      lineNum,
				UsageType: models.UsageReference,
			}
			a.addUsage(callback, usage)
			file.AddUsage(callback, usage)
		}
	}
}

func (a *UsageAnalyzer) detectCallableReferences(file *models.FileInfo, line string, lineNum int) {
	matches := patterns.CallableRefPattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) > 1 {
			funcName := match[1]
			if patterns.IsProtectedFunction(funcName) {
				continue
			}
			if !a.isKnownEntity(funcName) {
				continue
			}

			usage := models.Usage{
				Entity:    funcName,
				File:      file.RelativePath,
				Line:      lineNum,
				UsageType: models.UsageReference,
			}
			a.addUsage(funcName, usage)
			file.AddUsage(funcName, usage)
		}
	}
}

func (a *UsageAnalyzer) isKnownEntity(name string) bool {
	if _, exists := a.entities[name]; exists {
		return true
	}
	for key := range a.entities {
		if strings.HasSuffix(key, "."+name) {
			return true
		}
	}
	return false
}

func (a *UsageAnalyzer) detectConstantUsage(file *models.FileInfo, line string, lineNum int) {
	matches := patterns.ConstantRefPattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) > 0 {
			constName := match[0]

			// Track all constant-like identifiers (uppercase with underscores)
			// This ensures we catch usage even within the same file
			usage := models.Usage{
				Entity:    constName,
				File:      file.RelativePath,
				Line:      lineNum,
				UsageType: models.UsageReference,
			}
			a.addUsage(constName, usage)
			file.AddUsage(constName, usage)
		}
	}
}

func (a *UsageAnalyzer) detectEnumUsage(file *models.FileInfo, line string, lineNum int) {
	// Detect enum usage with dot notation (e.g., UIEvent.HEX_CLICKED)
	matches := patterns.EnumRefPattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) > 2 {
			enumName := match[1]
			valueName := match[2]

			enumUsage := models.Usage{
				Entity:    enumName,
				File:      file.RelativePath,
				Line:      lineNum,
				UsageType: models.UsageReference,
			}
			a.addUsage(enumName, enumUsage)
			file.AddUsage(enumName, enumUsage)

			fullName := fmt.Sprintf("%s.%s", enumName, valueName)
			valueUsage := models.Usage{
				Entity:    fullName,
				File:      file.RelativePath,
				Line:      lineNum,
				UsageType: models.UsageReference,
			}
			a.addUsage(fullName, valueUsage)
			file.AddUsage(fullName, valueUsage)
		}
	}

	// Detect enum/class usage in type annotations (e.g., event_type: UIEvent)
	typeMatches := patterns.TypeAnnotationPattern.FindAllStringSubmatch(line, -1)
	for _, match := range typeMatches {
		if len(match) > 1 {
			typeName := match[1]

			// Check if this type name is one of our tracked entities
			if _, exists := a.entities[typeName]; exists {
				usage := models.Usage{
					Entity:    typeName,
					File:      file.RelativePath,
					Line:      lineNum,
					UsageType: models.UsageReference,
				}
				a.addUsage(typeName, usage)
				file.AddUsage(typeName, usage)
			}
		}
	}
}

func (a *UsageAnalyzer) detectStringReferences(file *models.FileInfo, line string, lineNum int) {
	matches := patterns.StringLiteralPattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) > 1 {
			content := match[1]

			for name, entities := range a.entities {
				matched := strings.Contains(content, name)
				if !matched {
					for _, entity := range entities {
						if entity.Name != name && strings.Contains(content, entity.Name) {
							matched = true
							break
						}
					}
				}
				if matched {
					usage := models.Usage{
						Entity:    name,
						File:      file.RelativePath,
						Line:      lineNum,
						UsageType: models.UsageString,
					}
					a.addUsage(name, usage)
					file.AddUsage(name, usage)
				}
			}
		}
	}
}

func (a *UsageAnalyzer) addUsage(entity string, usage models.Usage) {
	a.usages[entity] = append(a.usages[entity], usage)
}

func (a *UsageAnalyzer) GetUsages(entity string) []models.Usage {
	return a.usages[entity]
}

func (a *UsageAnalyzer) GetUsageCount(entity string) int {
	return len(a.usages[entity])
}

func (a *UsageAnalyzer) IsUsed(entity *models.Entity) bool {
	if entity.IsProtected() {
		return true
	}

	usageCount := a.GetUsageCount(entity.Name)

	// Check by full name first (for enum values, etc.)
	fullName := entity.FullName()
	fullUsageCount := a.GetUsageCount(fullName)
	if fullUsageCount > 0 {
		return true
	}

	// Special handling for functions - check if the only usage is within the function itself
	if entity.Type == models.EntityFunction {
		if usageCount == 0 {
			return false
		}
		if usageCount == 1 {
			usages := a.GetUsages(entity.Name)
			if len(usages) > 0 && usages[0].File == entity.File &&
				usages[0].Line >= entity.Line && usages[0].Line <= entity.EndLine {
				// The only usage is within the function definition itself
				return false
			}
		}
		return true
	}

	// For all other entities (constants, signals, enums, class names), any usage counts
	return usageCount > 0
}
