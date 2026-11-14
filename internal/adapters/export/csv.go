package export

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"hw2kpo/internal/adapters/files"
	"hw2kpo/internal/application"
)

type CSVExporter struct{ storagePath string }

func NewCSVExporter(storagePath string) *CSVExporter { return &CSVExporter{storagePath: storagePath} }

func (e *CSVExporter) Export(rows []application.ResultRow, destination string) error {
	if strings.TrimSpace(destination) == "" {
		return fmt.Errorf("нужен путь файла экспорта")
	}
	target, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	storage, err := filepath.Abs(e.storagePath)
	if err != nil {
		return err
	}
	same := target == storage || (runtime.GOOS == "windows" && strings.EqualFold(target, storage))

	if a, errA := os.Stat(target); errA == nil {
		if b, errB := os.Stat(storage); errB == nil && os.SameFile(a, b) {
			same = true
		}
	}
	if same {
		return fmt.Errorf("нельзя экспортировать поверх файла хранилища")
	}
	return files.Write(target, func(w io.Writer) error {
		if _, err := io.WriteString(w, "\xef\xbb\xbf"); err != nil {
			return err
		}
		writer := csv.NewWriter(w)
		writer.Comma = ';'
		writer.UseCRLF = true
		if err := writer.Write([]string{"student_id", "student_name", "final_score", "assessed", "total", "policy"}); err != nil {
			return err
		}
		for _, row := range rows {
			if err := writer.Write([]string{
				row.StudentID, row.StudentName, strconv.FormatFloat(row.Score, 'f', 2, 64),
				strconv.Itoa(row.Assessed), strconv.Itoa(row.Total), row.Policy,
			}); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	})
}

var _ application.ResultsExporter = (*CSVExporter)(nil)
