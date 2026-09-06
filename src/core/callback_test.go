package core

import (
	"github.com/aviorstudio/gdlint/src/models"
	"os"
	"path/filepath"
	"testing"
)

func TestUnusedAnalysisKeepsEngineCallbacks(t *testing.T) {
	root := t.TempDir()
	script := "@tool\nextends Control\n\nfunc _get_configuration_warnings() -> PackedStringArray:\n\treturn PackedStringArray()\n\nfunc _get_property_list() -> Array:\n\treturn []\n"
	if err := os.WriteFile(filepath.Join(root, "example.gd"), []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	config := &models.LintConfig{Errors: models.ErrorSection{UnusedFunctions: true}}
	results, err := NewAnalyzer(root, config).Analyze()
	if err != nil {
		t.Fatal(err)
	}
	if len(results.UnusedFunctions) != 0 {
		t.Fatalf("engine callbacks reported unused: %+v", results.UnusedFunctions)
	}
}
