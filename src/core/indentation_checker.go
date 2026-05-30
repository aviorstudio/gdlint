package core

import (
	"strings"

	"github.com/aviorstudio/gdlint/src/models"
)

type IndentationChecker struct {
	files []*models.FileInfo
}

func NewIndentationChecker(files []*models.FileInfo) *IndentationChecker {
	return &IndentationChecker{files: files}
}

func (c *IndentationChecker) CheckIndentation() []models.CodeLocation {
	var issues []models.CodeLocation
	
	for _, file := range c.files {
		fileIssues := c.checkFile(file)
		issues = append(issues, fileIssues...)
	}
	
	return issues
}

func (c *IndentationChecker) checkFile(file *models.FileInfo) []models.CodeLocation {
	var issues []models.CodeLocation
	
	// GDScript should use tabs for indentation
	for i, line := range file.Lines {
		lineNum := i + 1
		
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		// Get leading whitespace
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" {
			// Line is all whitespace
			continue
		}
		
		indent := line[:len(line)-len(trimmed)]
		
		// Check for spaces in indentation
		if strings.Contains(indent, " ") {
			// Check if it's mixed (both tabs and spaces)
			if strings.Contains(indent, "\t") {
				issues = append(issues, models.CodeLocation{
					File:   file.RelativePath,
					Line:   lineNum,
					Column: 0,
					Text:   "Mixed tabs and spaces in indentation",
				})
			} else {
				// Pure spaces - this is wrong for GDScript
				issues = append(issues, models.CodeLocation{
					File:   file.RelativePath,
					Line:   lineNum,
					Column: 0,
					Text:   "Used spaces for indentation (GDScript requires tabs)",
				})
			}
		}
	}
	
	return issues
}

