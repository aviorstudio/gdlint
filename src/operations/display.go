package operations

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aviorstudio/gdlint/src/models"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
	ColorBold   = "\033[1m"
)

type Display struct{}

func NewDisplay() *Display {
	return &Display{}
}

func (d *Display) ShowResults(results *models.AnalysisResults, showWarnings bool, listLimit int) bool {
	hasErrors := results.HasErrors()

	if hasErrors {
		d.showErrors(results, listLimit)
	}

	if showWarnings && results.HasWarnings() {
		d.showWarnings(&results.Warnings, listLimit)
	}

	if !hasErrors && !results.HasWarnings() {
		fmt.Printf("%s✅ No issues found - codebase is clean!%s\n", ColorGreen, ColorReset)
		return false
	}

	d.showSummary(results, showWarnings)

	return hasErrors
}

func (d *Display) showErrors(results *models.AnalysisResults, listLimit int) {
	fmt.Printf("%s%s=== ERRORS (will be removed with -fix) ===%s\n\n", ColorBold, ColorRed, ColorReset)

	if len(results.UnusedFunctions) > 0 {
		d.showEntityList("Unused functions", results.UnusedFunctions, listLimit, ColorRed)
	}

	if len(results.UnusedSignals) > 0 {
		d.showEntityList("Unused signals", results.UnusedSignals, listLimit, ColorRed)
	}

	if len(results.UnusedConstants) > 0 {
		d.showEntityList("Unused constants", results.UnusedConstants, listLimit, ColorRed)
	}

	if len(results.UnusedEnums) > 0 {
		d.showEntityList("Unused enums", results.UnusedEnums, listLimit, ColorRed)
	}

	if len(results.UnusedEnumValues) > 0 {
		d.showEntityList("Unused enum values", results.UnusedEnumValues, listLimit, ColorRed)
	}

	if len(results.UnusedClassNames) > 0 {
		d.showEntityList("Unused class names", results.UnusedClassNames, listLimit, ColorRed)
	}

	if len(results.UnusedFiles) > 0 {
		d.showFileList("Unused files", results.UnusedFiles, listLimit, ColorRed)
	}

	if len(results.PrintStatements) > 0 {
		d.showLocationList("Print statements", results.PrintStatements, listLimit, ColorRed)
	}

	if len(results.Comments) > 0 {
		d.showLocationList("Comments to remove", results.Comments, listLimit, ColorRed)
	}

	if len(results.PassStatements) > 0 {
		d.showLocationList("Unnecessary pass statements", results.PassStatements, listLimit, ColorRed)
	}

	if len(results.OrphanedUIDs) > 0 {
		d.showFileList("Orphaned UID files", results.OrphanedUIDs, listLimit, ColorRed)
	}

	if len(results.IndentationIssues) > 0 {
		d.showLocationList("Indentation issues", results.IndentationIssues, listLimit, ColorRed)
	}

	if len(results.HasMethodCalls) > 0 {
		fmt.Printf("%s%s❌ has_method() calls (ALWAYS ERRORS):%s %d\n", ColorBold, ColorRed, ColorReset, len(results.HasMethodCalls))
		d.showLocationDetails(results.HasMethodCalls, listLimit)
		fmt.Println()
	}

	if len(results.VariantUsages) > 0 {
		d.showLocationList("Explicit Variant usages (disallowed once enabled)", results.VariantUsages, listLimit, ColorRed)
	}

	if len(results.MissingReturnTypes) > 0 {
		d.showLocationList("Missing return type annotations", results.MissingReturnTypes, listLimit, ColorRed)
	}
}

func (d *Display) showWarnings(warnings *models.Warnings, listLimit int) {
	fmt.Printf("%s%s=== WARNINGS ===%s\n\n", ColorBold, ColorYellow, ColorReset)

	if len(warnings.SingleUseFunctions) > 0 {
		d.showEntityList("Single-use functions", warnings.SingleUseFunctions, listLimit, ColorYellow)
	}

	if len(warnings.EmptyFunctions) > 0 {
		d.showEntityList("Empty functions", warnings.EmptyFunctions, listLimit, ColorYellow)
	}
}

func (d *Display) showEntityList(title string, entities []*models.Entity, limit int, color string) {
	fmt.Printf("%s%s:%s %d\n", color, title, ColorReset, len(entities))

	sort.Slice(entities, func(i, j int) bool {
		if entities[i].File != entities[j].File {
			return entities[i].File < entities[j].File
		}
		return entities[i].Line < entities[j].Line
	})

	count := len(entities)
	if limit > 0 && limit < count {
		count = limit
	}

	for i := 0; i < count; i++ {
		entity := entities[i]
		fmt.Printf("  %s• %s:%d: %s%s\n", ColorGray, entity.File, entity.Line, entity.Name, ColorReset)
	}

	if limit > 0 && limit < len(entities) {
		fmt.Printf("  %s... and %d more%s\n", ColorGray, len(entities)-limit, ColorReset)
	}

	fmt.Println()
}

func (d *Display) showFileList(title string, files []string, limit int, color string) {
	fmt.Printf("%s%s:%s %d\n", color, title, ColorReset, len(files))

	sort.Strings(files)

	count := len(files)
	if limit > 0 && limit < count {
		count = limit
	}

	for i := 0; i < count; i++ {
		fmt.Printf("  %s• %s%s\n", ColorGray, files[i], ColorReset)
	}

	if limit > 0 && limit < len(files) {
		fmt.Printf("  %s... and %d more%s\n", ColorGray, len(files)-limit, ColorReset)
	}

	fmt.Println()
}

func (d *Display) showLocationList(title string, locations []models.CodeLocation, limit int, color string) {
	fmt.Printf("%s%s:%s %d\n", color, title, ColorReset, len(locations))

	sort.Slice(locations, func(i, j int) bool {
		if locations[i].File != locations[j].File {
			return locations[i].File < locations[j].File
		}
		return locations[i].Line < locations[j].Line
	})

	count := len(locations)
	if limit > 0 && limit < count {
		count = limit
	}

	for i := 0; i < count; i++ {
		loc := locations[i]
		preview := loc.Text
		if len(preview) > 60 {
			preview = preview[:60] + "..."
		}
		fmt.Printf("  %s• %s:%d: %s%s\n", ColorGray, loc.File, loc.Line, preview, ColorReset)
	}

	if limit > 0 && limit < len(locations) {
		fmt.Printf("  %s... and %d more%s\n", ColorGray, len(locations)-limit, ColorReset)
	}

	fmt.Println()
}

func (d *Display) showLocationDetails(locations []models.CodeLocation, limit int) {
	sort.Slice(locations, func(i, j int) bool {
		if locations[i].File != locations[j].File {
			return locations[i].File < locations[j].File
		}
		return locations[i].Line < locations[j].Line
	})

	count := len(locations)
	if limit > 0 && limit < count {
		count = limit
	}

	for i := 0; i < count; i++ {
		loc := locations[i]
		fmt.Printf("  %s• %s:%d: %s%s\n", ColorRed, loc.File, loc.Line, loc.Text, ColorReset)
	}

	if limit > 0 && limit < len(locations) {
		fmt.Printf("  %s... and %d more%s\n", ColorGray, len(locations)-limit, ColorReset)
	}
}

func (d *Display) showSummary(results *models.AnalysisResults, showWarnings bool) {
	fmt.Printf("%s%s=== SUMMARY ===%s\n", ColorBold, ColorCyan, ColorReset)

	errorCount := results.ErrorCount()
	warningCount := results.WarningCount()

	if errorCount > 0 {
		fmt.Printf("%s❌ %d errors found%s\n", ColorRed, errorCount, ColorReset)
		fmt.Printf("   Run with %s-fix%s to remove all errors\n", ColorBold, ColorReset)
	}

	if warningCount > 0 {
		if showWarnings {
			fmt.Printf("%s⚠️  %d warnings found%s\n", ColorYellow, warningCount, ColorReset)
		} else {
			fmt.Printf("%s⚠️  %d warnings found%s (run with %s-warn%s to see them)\n",
				ColorYellow, warningCount, ColorReset, ColorBold, ColorReset)
		}
	}

	if errorCount == 0 && warningCount == 0 {
		fmt.Printf("%s✅ No issues found!%s\n", ColorGreen, ColorReset)
	}
}

func (d *Display) formatFileLocation(file string, line int) string {
	return fmt.Sprintf("%s:%d", file, line)
}

func (d *Display) truncateText(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	if len(text) > maxLen {
		return text[:maxLen-3] + "..."
	}
	return text
}
