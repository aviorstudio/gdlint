package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aviorstudio/gdlint/src/models"
	"github.com/aviorstudio/gdlint/src/operations"
	"github.com/aviorstudio/gdlint/src/optimization"
	"github.com/aviorstudio/gdlint/src/patterns"
)

type Analyzer struct {
	rootDir         string
	config          *models.LintConfig
	files           []*models.FileInfo
	cache           *optimization.Cache
	autoloads       map[string]bool
	sceneFiles      map[string][]string
	referencedFiles map[string]bool
}

func NewAnalyzer(rootDir string, config *models.LintConfig) *Analyzer {
	return &Analyzer{
		rootDir:         rootDir,
		config:          config,
		files:           make([]*models.FileInfo, 0),
		cache:           optimization.NewCache(),
		autoloads:       make(map[string]bool),
		sceneFiles:      make(map[string][]string),
		referencedFiles: make(map[string]bool),
	}
}

func (a *Analyzer) Analyze() (*models.AnalysisResults, error) {
	if err := a.collectFiles(); err != nil {
		return nil, fmt.Errorf("failed to collect files: %w", err)
	}

	if err := a.parseFiles(); err != nil {
		return nil, fmt.Errorf("failed to parse files: %w", err)
	}

	a.detectAutoloads()
	a.scanSceneFiles()
	a.detectFileReferences()

	usageAnalyzer := NewUsageAnalyzer(a.files)
	usageAnalyzer.AnalyzeUsages()

	results := models.NewAnalysisResults()

	if a.config.Errors.UnusedFunctions {
		a.findUnusedFunctions(results, usageAnalyzer)
	}

	if a.config.Errors.UnusedSignals {
		a.findUnusedSignals(results, usageAnalyzer)
	}

	if a.config.Errors.UnusedConstants {
		a.findUnusedConstants(results, usageAnalyzer)
	}

	if a.config.Errors.UnusedEnums {
		a.findUnusedEnums(results, usageAnalyzer)
	}

	if a.config.Errors.UnusedClassNames {
		a.findUnusedClassNames(results, usageAnalyzer)
	}

	if a.config.Errors.UnusedFiles {
		a.findUnusedFiles(results)
	}

	if a.config.Errors.PrintStatements {
		a.findPrintStatements(results)
	}

	if a.config.Errors.Comments {
		a.findComments(results)
	}

	if a.config.Errors.PassStatements {
		passAnalyzer := NewPassAnalyzer(a.files)
		results.PassStatements = passAnalyzer.FindUnnecessaryPass()
	}

	if a.config.Errors.Indentation {
		indentChecker := NewIndentationChecker(a.files)
		results.IndentationIssues = indentChecker.CheckIndentation()
	}

	if a.config.Errors.HasMethod {
		a.findHasMethodCalls(results)
	}

	if a.config.Errors.OrphanedUIDs {
		a.findOrphanedUIDs(results)
	}

	if a.config.Errors.VariantUsage {
		a.findVariantUsages(results)
	}

	if a.config.Errors.MissingReturnTypes {
		checker := NewReturnTypeChecker(a.files)
		results.MissingReturnTypes = checker.Check()
	}

	if a.config.Warnings.SingleUseFunctions {
		a.findSingleUseFunctions(results, usageAnalyzer)
	}

	if a.config.Warnings.EmptyFunctions {
		a.findEmptyFunctions(results)
	}

	return results, nil
}

func (a *Analyzer) collectFiles() error {
	return filepath.Walk(a.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Skip generated build artifacts, but keep gameplay routes like src/routes/build.
		if a.isBuildArtifact(path) {
			return nil
		}

		if strings.HasSuffix(path, ".gd") {
			if a.shouldIgnore(path) {
				return nil
			}

			fileInfo := models.NewFileInfo(path, a.rootDir)
			a.files = append(a.files, fileInfo)
		}

		return nil
	})
}

func (a *Analyzer) shouldIgnore(path string) bool {
	for _, pattern := range a.config.Settings.IgnorePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

func (a *Analyzer) shouldIgnoreVariant(path string) bool {
	for _, pattern := range a.config.Settings.VariantIgnorePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

func (a *Analyzer) shouldIgnoreUnused(path string) bool {
	for _, pattern := range a.config.Settings.UnusedIgnorePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

func (a *Analyzer) isBuildArtifact(path string) bool {
	relPath, err := filepath.Rel(a.rootDir, path)
	if err != nil {
		relPath = path
	}

	parts := strings.Split(relPath, string(os.PathSeparator))
	foundSrc := false
	for _, part := range parts {
		if part == "src" {
			foundSrc = true
		}
		if part == "build" {
			// Only skip build artifacts that live outside src/.
			if !foundSrc {
				return true
			}
		}
	}
	return false
}

func (a *Analyzer) parseFiles() error {
	fileOps := operations.NewFileOperations(a.rootDir)

	for _, file := range a.files {
		content, err := fileOps.ReadFile(file.Path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file.Path, err)
		}

		file.Content = content
		file.Lines = strings.Split(content, "\n")

		detector := NewEntityDetector(file)
		detector.DetectEntities()
	}

	return nil
}

func (a *Analyzer) detectAutoloads() {
	// Use the new file reference detector
	autoloadBasenames, autoloadPaths := FindAutoloadFiles(a.rootDir)
	a.autoloads = autoloadBasenames

	// Mark autoload files as referenced
	for path := range autoloadPaths {
		a.referencedFiles[path] = true
	}

	// Mark autoload files
	for _, file := range a.files {
		baseName := filepath.Base(file.Path)
		if autoloadBasenames[baseName] {
			file.IsAutoload = true
		}
	}
}

func (a *Analyzer) scanSceneFiles() {
	// Use the new file reference detector to process scene files
	sceneRefs := ProcessSceneFiles(a.rootDir)
	for ref := range sceneRefs {
		a.referencedFiles[ref] = true
	}
}

func (a *Analyzer) detectFileReferences() {
	// Collect all .gd file paths
	var gdFiles []string
	for _, file := range a.files {
		gdFiles = append(gdFiles, file.Path)
	}

	// Process file references (preload, load, extends)
	fileRefs := ProcessFileReferences(gdFiles, a.rootDir)
	for ref := range fileRefs {
		a.referencedFiles[ref] = true
	}
}

func (a *Analyzer) findUnusedFunctions(results *models.AnalysisResults, usageAnalyzer *UsageAnalyzer) {
	for _, file := range a.files {
		if a.shouldIgnoreUnused(file.Path) {
			continue
		}
		for _, entity := range file.GetEntitiesByType(models.EntityFunction) {
			if !usageAnalyzer.IsUsed(entity) && !entity.IsProtected() {
				results.UnusedFunctions = append(results.UnusedFunctions, entity)
			}
		}
	}
}

func (a *Analyzer) findUnusedSignals(results *models.AnalysisResults, usageAnalyzer *UsageAnalyzer) {
	for _, file := range a.files {
		if a.shouldIgnoreUnused(file.Path) {
			continue
		}
		for _, entity := range file.GetEntitiesByType(models.EntitySignal) {
			if !usageAnalyzer.IsUsed(entity) {
				results.UnusedSignals = append(results.UnusedSignals, entity)
			}
		}
	}
}

func (a *Analyzer) findUnusedConstants(results *models.AnalysisResults, usageAnalyzer *UsageAnalyzer) {
	for _, file := range a.files {
		if a.shouldIgnoreUnused(file.Path) {
			continue
		}
		for _, entity := range file.GetEntitiesByType(models.EntityConstant) {
			if !usageAnalyzer.IsUsed(entity) {
				results.UnusedConstants = append(results.UnusedConstants, entity)
			}
		}
	}
}

func (a *Analyzer) findUnusedEnums(results *models.AnalysisResults, usageAnalyzer *UsageAnalyzer) {
	for _, file := range a.files {
		if a.shouldIgnoreUnused(file.Path) {
			continue
		}
		for _, entity := range file.GetEntitiesByType(models.EntityEnum) {
			if !usageAnalyzer.IsUsed(entity) {
				results.UnusedEnums = append(results.UnusedEnums, entity)
			}
		}

		for _, entity := range file.GetEntitiesByType(models.EntityEnumValue) {
			fullName := entity.FullName()
			if !usageAnalyzer.IsUsed(entity) && usageAnalyzer.GetUsageCount(fullName) == 0 {
				results.UnusedEnumValues = append(results.UnusedEnumValues, entity)
			}
		}
	}
}

func (a *Analyzer) findUnusedClassNames(results *models.AnalysisResults, usageAnalyzer *UsageAnalyzer) {
	// Class names are never considered unused - they're always protected
	// This matches the Python linter behavior
}

func (a *Analyzer) findUnusedFiles(results *models.AnalysisResults) {
	for _, file := range a.files {
		if a.shouldIgnoreUnused(file.Path) {
			continue
		}

		// Skip autoload files
		if file.IsAutoload {
			continue
		}

		// Get absolute path for comparison
		absPath, err := filepath.Abs(file.Path)
		if err != nil {
			absPath = file.Path
		}

		// Check if file is referenced
		if !a.referencedFiles[absPath] {
			// Also check if any class_name from this file is used
			hasUsedClassName := false
			for _, entity := range file.GetEntitiesByType(models.EntityClassName) {
				// Check if the class name is referenced anywhere
				for _, otherFile := range a.files {
					if otherFile.Path == file.Path {
						continue
					}
					if strings.Contains(otherFile.Content, entity.Name) {
						hasUsedClassName = true
						break
					}
				}
				if hasUsedClassName {
					break
				}
			}

			if !hasUsedClassName {
				results.UnusedFiles = append(results.UnusedFiles, file.RelativePath)
			}
		}
	}
}

func (a *Analyzer) findPrintStatements(results *models.AnalysisResults) {
	for _, file := range a.files {
		for i, line := range file.Lines {
			// Check for both print statements and debug log statements
			if patterns.PrintPattern.MatchString(line) || patterns.DebugLogPattern.MatchString(line) {
				// Check if this line has gdlint-ignore-print comment
				if patterns.IgnorePrintPattern.MatchString(line) {
					continue
				}

				// Check if previous line has gdlint-ignore-print comment
				if i > 0 && patterns.IgnorePrintPattern.MatchString(file.Lines[i-1]) {
					continue
				}

				results.PrintStatements = append(results.PrintStatements, models.CodeLocation{
					File:   file.RelativePath,
					Line:   i + 1,
					Column: 0,
					Text:   strings.TrimSpace(line),
				})
			}
		}
	}
}

func (a *Analyzer) findComments(results *models.AnalysisResults) {
	for _, file := range a.files {
		for i, line := range file.Lines {
			if patterns.CommentPattern.MatchString(line) {
				// Always allow gdlint-ignore comments
				if patterns.IgnorePrintPattern.MatchString(line) {
					continue
				}

				isAllowed := false
				for _, prefix := range a.config.Settings.AllowedCommentPrefixes {
					if strings.Contains(line, prefix) {
						isAllowed = true
						break
					}
				}

				if !isAllowed {
					results.Comments = append(results.Comments, models.CodeLocation{
						File:   file.RelativePath,
						Line:   i + 1,
						Column: 0,
						Text:   strings.TrimSpace(line),
					})
				}
			}
		}
	}
}

func (a *Analyzer) findHasMethodCalls(results *models.AnalysisResults) {
	for _, file := range a.files {
		for i, line := range file.Lines {
			if patterns.HasMethodPattern.MatchString(line) {
				results.HasMethodCalls = append(results.HasMethodCalls, models.CodeLocation{
					File:   file.RelativePath,
					Line:   i + 1,
					Column: 0,
					Text:   strings.TrimSpace(line),
				})
			}
		}
	}
}

func (a *Analyzer) findVariantUsages(results *models.AnalysisResults) {
	for _, file := range a.files {
		if a.shouldIgnoreVariant(file.Path) {
			continue
		}
		for i, line := range file.Lines {
			clean := patterns.RemoveComments(line)
			if clean == "" {
				continue
			}
			if patterns.VariantPattern.MatchString(clean) {
				results.VariantUsages = append(results.VariantUsages, models.CodeLocation{
					File:   file.RelativePath,
					Line:   i + 1,
					Column: 0,
					Text:   strings.TrimSpace(clean),
				})
			}
		}
	}
}

func (a *Analyzer) findOrphanedUIDs(results *models.AnalysisResults) {
	err := filepath.Walk(a.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, ".gd.uid") {
			if a.shouldIgnore(path) {
				return nil
			}

			gdPath := strings.TrimSuffix(path, ".uid")
			if _, err := os.Stat(gdPath); os.IsNotExist(err) {
				relPath, _ := filepath.Rel(a.rootDir, path)
				results.OrphanedUIDs = append(results.OrphanedUIDs, relPath)
			}
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to scan for orphaned UIDs: %v\n", err)
	}
}

func (a *Analyzer) findSingleUseFunctions(results *models.AnalysisResults, usageAnalyzer *UsageAnalyzer) {
	for _, file := range a.files {
		for _, entity := range file.GetEntitiesByType(models.EntityFunction) {
			if entity.IsProtected() {
				continue
			}

			usageCount := usageAnalyzer.GetUsageCount(entity.Name)
			if usageCount == 1 {
				results.Warnings.SingleUseFunctions = append(results.Warnings.SingleUseFunctions, entity)
			}
		}
	}
}

func (a *Analyzer) findEmptyFunctions(results *models.AnalysisResults) {
	for _, file := range a.files {
		for _, entity := range file.GetEntitiesByType(models.EntityFunction) {
			if entity.IsProtected() {
				continue
			}

			isEmpty := true
			// Start from line after the function declaration
			// For functions at EOF, EndLine can be len(file.Lines)+1, so we need <= instead of <
			for i := entity.Line + 1; i <= entity.EndLine && i <= len(file.Lines); i++ {
				line := file.Lines[i-1]
				cleanLine := patterns.RemoveComments(line)
				trimmed := strings.TrimSpace(cleanLine)

				// Check if there's any meaningful content (not just pass or empty)
				if trimmed != "" && trimmed != "pass" {
					isEmpty = false
					break
				}
			}

			if isEmpty {
				results.Warnings.EmptyFunctions = append(results.Warnings.EmptyFunctions, entity)
			}
		}
	}
}
