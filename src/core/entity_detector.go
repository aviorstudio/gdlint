package core

import (
	"strings"

	"github.com/aviorstudio/gdlint/src/models"
	"github.com/aviorstudio/gdlint/src/patterns"
)

type EntityDetector struct {
	file *models.FileInfo
}

func NewEntityDetector(file *models.FileInfo) *EntityDetector {
	return &EntityDetector{file: file}
}

func (d *EntityDetector) DetectEntities() {
	lines := strings.Split(d.file.Content, "\n")
	d.file.Lines = lines
	
	var currentEnum string
	var enumIndent int
	inEnum := false
	var lastRPCLine int
	var className string
	
	for i, line := range lines {
		lineNum := i + 1
		
		if matches := patterns.ClassNamePattern.FindStringSubmatch(line); len(matches) > 1 {
			className = matches[1]
			d.file.ScriptClass = className
			entity := &models.Entity{
				Type:   models.EntityClassName,
				Name:   className,
				File:   d.file.RelativePath,
				Line:   lineNum,
				Column: 0,
			}
			d.file.AddEntity(entity)
		}
		
		if patterns.RPCPattern.MatchString(line) {
			lastRPCLine = lineNum
		}
		
		if matches := patterns.FunctionPattern.FindStringSubmatch(line); len(matches) > 2 {
			indent := matches[1]
			funcName := matches[2]
			isStatic := patterns.StaticFuncPattern.MatchString(line)
			isRPC := (lastRPCLine == lineNum - 1)
			
			entity := &models.Entity{
				Type:        models.EntityFunction,
				Name:        funcName,
				File:        d.file.RelativePath,
				Line:        lineNum,
				Column:      len(indent),
				IndentLevel: patterns.CountIndentLevel(indent),
				IsStatic:    isStatic,
				IsRPC:       isRPC,
				Parent:      className,
			}
			
			endLine := d.findFunctionEnd(lines, i, entity.IndentLevel)
			entity.EndLine = endLine
			
			d.file.AddEntity(entity)
		}
		
		if matches := patterns.SignalPattern.FindStringSubmatch(line); len(matches) > 2 {
			indent := matches[1]
			signalName := matches[2]
			
			entity := &models.Entity{
				Type:        models.EntitySignal,
				Name:        signalName,
				File:        d.file.RelativePath,
				Line:        lineNum,
				Column:      len(indent),
				IndentLevel: patterns.CountIndentLevel(indent),
				Parent:      className,
			}
			d.file.AddEntity(entity)
		}
		
		if matches := patterns.ConstantPattern.FindStringSubmatch(line); len(matches) > 2 {
			indent := matches[1]
			constName := matches[2]
			
			value := ""
			if idx := strings.Index(line, "="); idx > 0 {
				value = strings.TrimSpace(line[idx+1:])
			}
			
			entity := &models.Entity{
				Type:        models.EntityConstant,
				Name:        constName,
				File:        d.file.RelativePath,
				Line:        lineNum,
				Column:      len(indent),
				IndentLevel: patterns.CountIndentLevel(indent),
				Value:       value,
				Parent:      className,
			}
			d.file.AddEntity(entity)
		}
		
		if matches := patterns.EnumPattern.FindStringSubmatch(line); len(matches) > 2 {
			indent := matches[1]
			enumName := matches[2]
			
			entity := &models.Entity{
				Type:        models.EntityEnum,
				Name:        enumName,
				File:        d.file.RelativePath,
				Line:        lineNum,
				Column:      len(indent),
				IndentLevel: patterns.CountIndentLevel(indent),
				Parent:      className,
			}
			d.file.AddEntity(entity)
			
			currentEnum = enumName
			enumIndent = patterns.CountIndentLevel(indent)
			inEnum = true
		}
		
		if inEnum && strings.Contains(line, "}") {
			lineIndent := patterns.CountIndentLevel(patterns.ExtractIndentation(line))
			if lineIndent <= enumIndent {
				inEnum = false
				currentEnum = ""
			}
		}
		
		if inEnum && currentEnum != "" {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && trimmed != "{" && trimmed != "}" {
				if matches := patterns.EnumValuePattern.FindStringSubmatch(trimmed); len(matches) > 1 {
					valueName := matches[1]
					
					entity := &models.Entity{
						Type:        models.EntityEnumValue,
						Name:        valueName,
						File:        d.file.RelativePath,
						Line:        lineNum,
						Column:      0,
						IndentLevel: enumIndent + 1,
						Parent:      currentEnum,
					}
					d.file.AddEntity(entity)
				}
			}
		}
	}
}

func (d *EntityDetector) findFunctionEnd(lines []string, startIdx int, funcIndent int) int {
	for i := startIdx + 1; i < len(lines); i++ {
		line := lines[i]
		
		if patterns.IsEmptyOrComment(line) {
			continue
		}
		
		indent := patterns.ExtractIndentation(line)
		lineIndent := patterns.CountIndentLevel(indent)
		
		if lineIndent <= funcIndent && !patterns.IsEmptyOrComment(line) {
			return i
		}
	}
	
	return len(lines)
}