package core

import (
	"strings"

	"github.com/aviorstudio/gdlint/src/models"
	"github.com/aviorstudio/gdlint/src/patterns"
)

type PassAnalyzer struct {
	files []*models.FileInfo
}

func NewPassAnalyzer(files []*models.FileInfo) *PassAnalyzer {
	return &PassAnalyzer{files: files}
}

func (p *PassAnalyzer) FindUnnecessaryPass() []models.CodeLocation {
	var unnecessary []models.CodeLocation
	
	for _, file := range p.files {
		filePass := p.analyzeFile(file)
		unnecessary = append(unnecessary, filePass...)
	}
	
	return unnecessary
}

func (p *PassAnalyzer) analyzeFile(file *models.FileInfo) []models.CodeLocation {
	var unnecessary []models.CodeLocation
	
	for i, line := range file.Lines {
		lineNum := i + 1
		
		if !patterns.PassPattern.MatchString(line) {
			continue
		}
		
		if p.isUnnecessary(file.Lines, i) {
			unnecessary = append(unnecessary, models.CodeLocation{
				File:   file.RelativePath,
				Line:   lineNum,
				Column: 0,
				Text:   strings.TrimSpace(line),
			})
		}
	}
	
	return unnecessary
}

func (p *PassAnalyzer) isUnnecessary(lines []string, passIndex int) bool {
	currentIndent := patterns.CountIndentLevel(patterns.ExtractIndentation(lines[passIndex]))
	
	hasOtherContent := false
	for i := passIndex - 1; i >= 0; i-- {
		line := lines[i]
		
		if patterns.IsEmptyOrComment(line) {
			continue
		}
		
		lineIndent := patterns.CountIndentLevel(patterns.ExtractIndentation(line))
		
		if lineIndent < currentIndent {
			break
		}
		
		if lineIndent == currentIndent && !p.isBlockStart(line) {
			hasOtherContent = true
			break
		}
	}
	
	for i := passIndex + 1; i < len(lines); i++ {
		line := lines[i]
		
		if patterns.IsEmptyOrComment(line) {
			continue
		}
		
		lineIndent := patterns.CountIndentLevel(patterns.ExtractIndentation(line))
		
		if lineIndent < currentIndent {
			break
		}
		
		if lineIndent == currentIndent {
			hasOtherContent = true
			break
		}
	}
	
	return hasOtherContent
}

func (p *PassAnalyzer) isBlockStart(line string) bool {
	trimmed := strings.TrimSpace(line)
	
	blockStarters := []string{
		"func ", "if ", "elif ", "else:", "for ", "while ",
		"match ", "class ", "enum ",
	}
	
	for _, starter := range blockStarters {
		if strings.HasPrefix(trimmed, starter) {
			return true
		}
	}
	
	return strings.HasSuffix(trimmed, ":")
}