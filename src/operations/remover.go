package operations

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aviorstudio/gdlint/src/models"
	"github.com/aviorstudio/gdlint/src/patterns"
)

type Remover struct {
	rootDir string
	fileOps *FileOperations
}

func NewRemover(rootDir string) *Remover {
	return &Remover{
		rootDir: rootDir,
		fileOps: NewFileOperations(rootDir),
	}
}

func (r *Remover) RemoveAll(results *models.AnalysisResults) error {
	filesToModify := make(map[string][]removalTask)
	
	for _, entity := range results.UnusedFunctions {
		fullPath := filepath.Join(r.rootDir, entity.File)
		filesToModify[fullPath] = append(filesToModify[fullPath], removalTask{
			startLine: entity.Line,
			endLine:   entity.EndLine,
			taskType:  "function",
			name:      entity.Name,
		})
	}
	
	for _, entity := range results.UnusedSignals {
		fullPath := filepath.Join(r.rootDir, entity.File)
		filesToModify[fullPath] = append(filesToModify[fullPath], removalTask{
			startLine: entity.Line,
			endLine:   entity.Line,
			taskType:  "signal",
			name:      entity.Name,
		})
	}
	
	for _, entity := range results.UnusedConstants {
		fullPath := filepath.Join(r.rootDir, entity.File)
		filesToModify[fullPath] = append(filesToModify[fullPath], removalTask{
			startLine: entity.Line,
			endLine:   entity.Line,
			taskType:  "constant",
			name:      entity.Name,
		})
	}
	
	for _, entity := range results.UnusedEnums {
		fullPath := filepath.Join(r.rootDir, entity.File)
		endLine := r.findEnumEnd(fullPath, entity.Line)
		filesToModify[fullPath] = append(filesToModify[fullPath], removalTask{
			startLine: entity.Line,
			endLine:   endLine,
			taskType:  "enum",
			name:      entity.Name,
		})
	}
	
	for _, entity := range results.UnusedClassNames {
		fullPath := filepath.Join(r.rootDir, entity.File)
		filesToModify[fullPath] = append(filesToModify[fullPath], removalTask{
			startLine: entity.Line,
			endLine:   entity.Line,
			taskType:  "class_name",
			name:      entity.Name,
		})
	}
	
	for _, location := range results.PrintStatements {
		fullPath := filepath.Join(r.rootDir, location.File)
		filesToModify[fullPath] = append(filesToModify[fullPath], removalTask{
			startLine: location.Line,
			endLine:   location.Line,
			taskType:  "print",
		})
	}
	
	for _, location := range results.Comments {
		fullPath := filepath.Join(r.rootDir, location.File)
		filesToModify[fullPath] = append(filesToModify[fullPath], removalTask{
			startLine: location.Line,
			endLine:   location.Line,
			taskType:  "comment",
		})
	}
	
	for _, location := range results.PassStatements {
		fullPath := filepath.Join(r.rootDir, location.File)
		filesToModify[fullPath] = append(filesToModify[fullPath], removalTask{
			startLine: location.Line,
			endLine:   location.Line,
			taskType:  "pass",
		})
	}
	
	for filePath, tasks := range filesToModify {
		if err := r.processFile(filePath, tasks); err != nil {
			return fmt.Errorf("failed to process %s: %w", filePath, err)
		}
	}
	
	for _, unusedFile := range results.UnusedFiles {
		fullPath := filepath.Join(r.rootDir, unusedFile)
		fmt.Printf("Removing unused file: %s\n", unusedFile)
		if err := r.fileOps.DeleteFile(fullPath); err != nil {
			return fmt.Errorf("failed to delete %s: %w", unusedFile, err)
		}
	}
	
	for _, orphanedUID := range results.OrphanedUIDs {
		fullPath := filepath.Join(r.rootDir, orphanedUID)
		fmt.Printf("Removing orphaned UID: %s\n", orphanedUID)
		if err := r.fileOps.DeleteFile(fullPath); err != nil {
			return fmt.Errorf("failed to delete %s: %w", orphanedUID, err)
		}
	}
	
	return nil
}

type removalTask struct {
	startLine int
	endLine   int
	taskType  string
	name      string
}

func (r *Remover) processFile(filePath string, tasks []removalTask) error {
	content, err := r.fileOps.ReadFile(filePath)
	if err != nil {
		return err
	}
	
	lines := strings.Split(content, "\n")
	
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].startLine > tasks[j].startLine
	})
	
	for _, task := range tasks {
		lines = r.removeLines(lines, task)
	}
	
	newContent := strings.Join(lines, "\n")
	
	cleanContent := r.cleanupEmptyLines(newContent)
	
	return r.fileOps.WriteFile(filePath, cleanContent)
}

func (r *Remover) removeLines(lines []string, task removalTask) []string {
	if task.startLine <= 0 || task.startLine > len(lines) {
		return lines
	}
	
	startIdx := task.startLine - 1
	endIdx := task.endLine
	if endIdx > len(lines) {
		endIdx = len(lines)
	}
	
	switch task.taskType {
	case "function":
		return r.removeFunction(lines, startIdx, endIdx)
	case "signal", "constant", "class_name":
		return r.removeSingleLine(lines, startIdx)
	case "enum":
		return r.removeEnum(lines, startIdx)
	case "print":
		return r.removePrintStatement(lines, startIdx)
	case "comment":
		return r.removeComment(lines, startIdx)
	case "pass":
		return r.removePass(lines, startIdx)
	default:
		return lines
	}
}

func (r *Remover) removeFunction(lines []string, startIdx, endIdx int) []string {
	for i := startIdx - 1; i >= 0; i-- {
		line := lines[i]
		if patterns.RPCPattern.MatchString(line) || patterns.ExportPattern.MatchString(line) {
			startIdx = i
		} else if !patterns.IsEmptyOrComment(line) {
			break
		}
	}
	
	needsPass := r.checkNeedsPass(lines, startIdx, endIdx)
	
	if needsPass {
		indent := patterns.ExtractIndentation(lines[startIdx])
		replacement := indent + "pass"
		
		newLines := make([]string, 0, len(lines)-((endIdx-1)-startIdx))
		newLines = append(newLines, lines[:startIdx]...)
		newLines = append(newLines, replacement)
		newLines = append(newLines, lines[endIdx:]...)
		return newLines
	}
	
	newLines := make([]string, 0, len(lines)-(endIdx-startIdx))
	newLines = append(newLines, lines[:startIdx]...)
	newLines = append(newLines, lines[endIdx:]...)
	return newLines
}

func (r *Remover) removeSingleLine(lines []string, idx int) []string {
	if idx < 0 || idx >= len(lines) {
		return lines
	}
	
	newLines := make([]string, 0, len(lines)-1)
	newLines = append(newLines, lines[:idx]...)
	newLines = append(newLines, lines[idx+1:]...)
	return newLines
}

func (r *Remover) removeEnum(lines []string, startIdx int) []string {
	endIdx := startIdx + 1
	braceCount := 1
	
	for i := startIdx + 1; i < len(lines); i++ {
		line := lines[i]
		if strings.Contains(line, "{") {
			braceCount++
		}
		if strings.Contains(line, "}") {
			braceCount--
			if braceCount == 0 {
				endIdx = i + 1
				break
			}
		}
	}
	
	newLines := make([]string, 0, len(lines)-(endIdx-startIdx))
	newLines = append(newLines, lines[:startIdx]...)
	newLines = append(newLines, lines[endIdx:]...)
	return newLines
}

func (r *Remover) removePrintStatement(lines []string, idx int) []string {
	line := lines[idx]
	indent := patterns.ExtractIndentation(line)
	trimmed := strings.TrimSpace(line)

	// Check if the entire line is just a print or debug log statement
	if patterns.PrintPattern.MatchString(trimmed) || patterns.DebugLogPattern.MatchString(trimmed) {
		// Check if removing this line would leave an empty block
		needsPass := false
		if idx > 0 {
			prevLine := strings.TrimSpace(lines[idx-1])
			// Check if previous line ends with a colon (indicating a block start)
			if strings.HasSuffix(prevLine, ":") {
				// Check if this is the only statement in the block
				if idx+1 < len(lines) {
					nextLine := lines[idx+1]
					nextIndent := patterns.ExtractIndentation(nextLine)
					// If next line has less or equal indentation, this is the only statement
					if len(nextIndent) <= len(indent) || strings.TrimSpace(nextLine) == "" {
						needsPass = true
					}
				} else {
					// This is the last line in the file after a block start
					needsPass = true
				}
			}
		}

		if needsPass {
			// Replace with pass statement at the same indentation
			lines[idx] = indent + "pass"
			return lines
		}

		// Otherwise, remove the entire line
		return r.removeSingleLine(lines, idx)
	}

	// Fallback: remove the entire line if it looks like a standalone print
	return r.removeSingleLine(lines, idx)
}

func (r *Remover) removeComment(lines []string, idx int) []string {
	line := lines[idx]
	
	if patterns.CommentPattern.MatchString(strings.TrimSpace(line)) {
		return r.removeSingleLine(lines, idx)
	}
	
	newLine := patterns.RemoveComments(line)
	if strings.TrimSpace(newLine) == "" {
		return r.removeSingleLine(lines, idx)
	}
	
	lines[idx] = newLine
	return lines
}

func (r *Remover) removePass(lines []string, idx int) []string {
	return r.removeSingleLine(lines, idx)
}

func (r *Remover) checkNeedsPass(lines []string, startIdx, endIdx int) bool {
	if startIdx == 0 {
		return false
	}
	
	for i := startIdx - 1; i >= 0; i-- {
		line := lines[i]
		if patterns.IsEmptyOrComment(line) {
			continue
		}
		
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, ":") {
			hasOtherContent := false
			baseIndent := patterns.CountIndentLevel(patterns.ExtractIndentation(line))
			
			for j := endIdx; j < len(lines); j++ {
				checkLine := lines[j]
				if patterns.IsEmptyOrComment(checkLine) {
					continue
				}
				
				checkIndent := patterns.CountIndentLevel(patterns.ExtractIndentation(checkLine))
				if checkIndent <= baseIndent {
					break
				}
				if checkIndent == baseIndent + 1 {
					hasOtherContent = true
					break
				}
			}
			
			return !hasOtherContent
		}
		
		break
	}
	
	return false
}

func (r *Remover) findEnumEnd(filePath string, startLine int) int {
	content, err := r.fileOps.ReadFile(filePath)
	if err != nil {
		return startLine
	}
	
	lines := strings.Split(content, "\n")
	braceCount := 0
	foundStart := false
	
	for i := startLine - 1; i < len(lines); i++ {
		line := lines[i]
		
		if !foundStart && strings.Contains(line, "{") {
			foundStart = true
			braceCount = 1
			continue
		}
		
		if foundStart {
			if strings.Contains(line, "{") {
				braceCount++
			}
			if strings.Contains(line, "}") {
				braceCount--
				if braceCount == 0 {
					return i + 1
				}
			}
		}
	}
	
	return len(lines)
}

func (r *Remover) cleanupEmptyLines(content string) string {
	lines := strings.Split(content, "\n")
	cleaned := make([]string, 0, len(lines))
	
	emptyCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			emptyCount++
			if emptyCount <= 2 {
				cleaned = append(cleaned, line)
			}
		} else {
			emptyCount = 0
			cleaned = append(cleaned, line)
		}
	}
	
	for len(cleaned) > 0 && strings.TrimSpace(cleaned[len(cleaned)-1]) == "" {
		cleaned = cleaned[:len(cleaned)-1]
	}
	
	return strings.Join(cleaned, "\n")
}