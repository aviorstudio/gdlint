package core

import (
	"regexp"
	"strings"

	"github.com/aviorstudio/gdlint/src/models"
)

var returnAnnotationPattern = regexp.MustCompile(`\)\s*->\s*[^:]+\s*:`)

// ReturnTypeChecker finds public GDScript functions missing return type annotations.
type ReturnTypeChecker struct {
	files []*models.FileInfo
}

func NewReturnTypeChecker(files []*models.FileInfo) *ReturnTypeChecker {
	return &ReturnTypeChecker{files: files}
}

func (c *ReturnTypeChecker) Check() []models.CodeLocation {
	var results []models.CodeLocation

	for _, file := range c.files {
		results = append(results, c.checkFile(file)...)
	}

	return results
}

func (c *ReturnTypeChecker) checkFile(file *models.FileInfo) []models.CodeLocation {
	var results []models.CodeLocation

	for _, entity := range file.Entities {
		if entity.Type != models.EntityFunction {
			continue
		}

		// Skip private/protected functions (starts with _, lifecycle callbacks, RPCs, etc.)
		if strings.HasPrefix(entity.Name, "_") || entity.IsProtected() {
			continue
		}

		if c.hasReturnAnnotation(file, entity) {
			continue
		}

		results = append(results, models.CodeLocation{
			File:   file.RelativePath,
			Line:   entity.Line,
			Column: 0,
			Text:   "public function '" + entity.Name + "' missing return type annotation",
		})
	}

	return results
}

func (c *ReturnTypeChecker) hasReturnAnnotation(file *models.FileInfo, entity *models.Entity) bool {
	if entity.Line < 1 || entity.Line > len(file.Lines) {
		return false
	}

	// Collect the full function signature (may span multiple lines)
	var sigParts []string
	parenBalance := 0
	for i := entity.Line - 1; i < len(file.Lines); i++ {
		line := file.Lines[i]
		sigParts = append(sigParts, strings.TrimSpace(line))
		parenBalance += strings.Count(line, "(") - strings.Count(line, ")")
		if parenBalance <= 0 && strings.Contains(line, ":") {
			break
		}
	}

	combined := strings.Join(sigParts, " ")
	return returnAnnotationPattern.MatchString(combined)
}
