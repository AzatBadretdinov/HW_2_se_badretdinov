package files_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"hw2kpo/internal/adapters/files"
)

func TestInterruptedWritePreservesOldFileAndCleansTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "course.json")
	if err := os.WriteFile(path, []byte("previous"), 0600); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("simulated write failure")
	err := files.Write(path, func(w io.Writer) error {
		if _, err := io.WriteString(w, "partial"); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("write failure was swallowed: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "previous" {
		t.Fatalf("old file changed: %q, %v", content, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary file leaked: %v, %v", entries, err)
	}
}
