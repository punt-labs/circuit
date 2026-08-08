package playbook

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestPositiveExamplesValidate(t *testing.T) {
	t.Parallel()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	examples := []string{
		filepath.Join(repoRoot, "examples", "minimal.yaml"),
		filepath.Join(repoRoot, "examples", "pr-watch.yaml"),
	}

	for _, example := range examples {
		example := example
		t.Run(filepath.Base(example), func(t *testing.T) {
			t.Parallel()
			document, err := ParseFile(example)
			if err != nil {
				t.Fatalf("parse example: %v", err)
			}
			result := Validate(document)
			if !result.OK() {
				t.Fatalf("validate example: %s", result.Error())
			}
		})
	}
}
