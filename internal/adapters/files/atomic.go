package files

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)


func Write(path string, write func(io.Writer) error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("создание каталога: %w", err)
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".pos-*")
	if err != nil {
		return fmt.Errorf("создание временного файла: %w", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := write(f); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return fmt.Errorf("замена файла %s: %w", path, err)
	}
	return nil
}
