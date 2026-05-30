package core

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// File reference patterns - updated to handle const assignments
	preloadPattern     = regexp.MustCompile(`(?:const\s+\w+\s*=\s*)?preload\s*\(\s*["']([^"']+\.gd)["']`)
	loadPattern        = regexp.MustCompile(`(?:const\s+\w+\s*=\s*)?load\s*\(\s*["']([^"']+\.gd)["']`)
	extendsFilePattern = regexp.MustCompile(`extends\s+["']([^"']+\.gd)["']`)

	// Scene file patterns
	extResourcePattern = regexp.MustCompile(`\[ext_resource\s+type="Script"[^\]]*path="res://([^"]+\.gd)"`)
	scriptPathPattern  = regexp.MustCompile(`script\s*=\s*"res://([^"]+\.gd)"`)

	// Autoload pattern - matches project.godot format
	autoloadPattern = regexp.MustCompile(`="?\*res://(.+\.gd)"`)
)

// ExtractFileReferences extracts all file references from GDScript code
func ExtractFileReferences(content string) []string {
	var refs []string
	seen := make(map[string]bool)

	// Find preload references
	for _, match := range preloadPattern.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 && !seen[match[1]] {
			refs = append(refs, match[1])
			seen[match[1]] = true
		}
	}

	// Find load references
	for _, match := range loadPattern.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 && !seen[match[1]] {
			refs = append(refs, match[1])
			seen[match[1]] = true
		}
	}

	// Find extends file references
	for _, match := range extendsFilePattern.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 && !seen[match[1]] {
			refs = append(refs, match[1])
			seen[match[1]] = true
		}
	}

	return refs
}

// ExtractSceneReferences extracts script references from .tscn files
func ExtractSceneReferences(content string) []string {
	var refs []string
	seen := make(map[string]bool)

	// Find ext_resource Script references
	for _, match := range extResourcePattern.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 && !seen[match[1]] {
			refs = append(refs, match[1])
			seen[match[1]] = true
		}
	}

	// Find direct script path references
	for _, match := range scriptPathPattern.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 && !seen[match[1]] {
			refs = append(refs, match[1])
			seen[match[1]] = true
		}
	}

	return refs
}

// FindAutoloadFiles finds all autoloaded scripts from a project's project.godot file.
// Returns both basenames (for checking) and full paths (for marking as referenced)
func FindAutoloadFiles(projectRoot string) (map[string]bool, map[string]bool) {
	autoloadBasenames := make(map[string]bool)
	autoloadPaths := make(map[string]bool)

	projectFile := filepath.Join(projectRoot, "project.godot")
	if content, err := os.ReadFile(projectFile); err == nil {
		for _, match := range autoloadPattern.FindAllStringSubmatch(string(content), -1) {
			if len(match) > 1 {
				autoloadFile := filepath.Base(match[1])
				autoloadBasenames[autoloadFile] = true

				fullPath := filepath.Join(projectRoot, match[1])
				if absPath, err := filepath.Abs(fullPath); err == nil {
					autoloadPaths[absPath] = true
				}
			}
		}
	}

	return autoloadBasenames, autoloadPaths
}

// ResolvePath resolves a file reference to an absolute path
func ResolvePath(refPath, sourceFile string, projectRoot string) string {
	// Handle relative paths
	if strings.HasPrefix(refPath, "./") || strings.HasPrefix(refPath, "../") {
		currentDir := filepath.Dir(sourceFile)
		resolved := filepath.Join(currentDir, refPath)
		resolved = filepath.Clean(resolved)
		if _, err := os.Stat(resolved); err == nil {
			if abs, err := filepath.Abs(resolved); err == nil {
				return abs
			}
		}
	}

	// Handle res:// paths
	if strings.HasPrefix(refPath, "res://") {
		refPath = refPath[6:]
	}

	testPath := filepath.Join(projectRoot, refPath)
	if _, err := os.Stat(testPath); err == nil {
		if abs, err := filepath.Abs(testPath); err == nil {
			return abs
		}
	}

	return ""
}

// ProcessFileReferences processes all file references in the codebase
func ProcessFileReferences(files []string, projectRoot string) map[string]bool {
	referencedFiles := make(map[string]bool)

	for _, file := range files {
		if content, err := os.ReadFile(file); err == nil {
			refs := ExtractFileReferences(string(content))
			for _, ref := range refs {
				if resolved := ResolvePath(ref, file, projectRoot); resolved != "" {
					referencedFiles[resolved] = true
				}
			}
		}
	}

	return referencedFiles
}

// ProcessSceneFiles processes all .tscn files for script references
func ProcessSceneFiles(projectRoot string) map[string]bool {
	referencedFiles := make(map[string]bool)

	filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip build directories
		if shouldSkipBuildPath(path) {
			return nil
		}

		if strings.HasSuffix(path, ".tscn") {
			if content, err := os.ReadFile(path); err == nil {
				refs := ExtractSceneReferences(string(content))
				for _, ref := range refs {
					if resolved := ResolvePath(ref, path, projectRoot); resolved != "" {
						referencedFiles[resolved] = true
					}
				}
			}
		}

		return nil
	})

	return referencedFiles
}

func shouldSkipBuildPath(path string) bool {
	parts := strings.Split(path, string(os.PathSeparator))
	foundSrc := false
	for _, part := range parts {
		if part == "src" {
			foundSrc = true
		}
		if part == "build" && !foundSrc {
			return true
		}
	}
	return false
}
