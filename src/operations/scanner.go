package operations

import (
	"strings"

	"github.com/aviorstudio/gdlint/src/models"
	"github.com/aviorstudio/gdlint/src/patterns"
)

type Scanner struct {
	rootDir string
	fileOps *FileOperations
}

func NewScanner(rootDir string) *Scanner {
	return &Scanner{
		rootDir: rootDir,
		fileOps: NewFileOperations(rootDir),
	}
}

func (s *Scanner) FindTechDebtComments() ([]models.CodeLocation, error) {
	var comments []models.CodeLocation

	files, err := s.fileOps.GetGDFiles(s.rootDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		content, err := s.fileOps.ReadFile(file)
		if err != nil {
			continue
		}

		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if matches := patterns.TechDebtPattern.FindStringSubmatch(line); len(matches) > 1 {
				comments = append(comments, models.CodeLocation{
					File:   s.fileOps.GetRelativePath(file),
					Line:   i + 1,
					Column: 0,
					Text:   strings.TrimSpace(matches[1]),
				})
			}
		}
	}

	return comments, nil
}
