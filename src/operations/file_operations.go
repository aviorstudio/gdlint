package operations

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileOperations struct {
	rootDir string
}

func NewFileOperations(rootDir string) *FileOperations {
	return &FileOperations{rootDir: rootDir}
}

func (f *FileOperations) ReadFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (f *FileOperations) WriteFile(path string, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	return os.WriteFile(path, []byte(content), 0644)
}

func (f *FileOperations) DeleteFile(path string) error {
	return os.Remove(path)
}

func (f *FileOperations) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (f *FileOperations) GetGDFiles(dir string) ([]string, error) {
	var files []string
	
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() && strings.HasSuffix(path, ".gd") {
			files = append(files, path)
		}
		
		return nil
	})
	
	return files, err
}

func (f *FileOperations) GetRelativePath(path string) string {
	rel, err := filepath.Rel(f.rootDir, path)
	if err != nil {
		return path
	}
	return rel
}

func (f *FileOperations) BackupFile(path string) error {
	content, err := f.ReadFile(path)
	if err != nil {
		return err
	}
	
	backupPath := path + ".backup"
	return f.WriteFile(backupPath, content)
}

func (f *FileOperations) RestoreFile(path string) error {
	backupPath := path + ".backup"
	
	if !f.FileExists(backupPath) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}
	
	content, err := f.ReadFile(backupPath)
	if err != nil {
		return err
	}
	
	if err := f.WriteFile(path, content); err != nil {
		return err
	}
	
	return f.DeleteFile(backupPath)
}

func (f *FileOperations) FindFiles(pattern string) ([]string, error) {
	var matches []string
	
	err := filepath.Walk(f.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			matches = append(matches, path)
		}
		
		return nil
	})
	
	return matches, err
}