package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runUI(t *testing.T, data, destination, input string) string {
	t.Helper()
	var out bytes.Buffer
	ui, err := compose(data, destination, strings.NewReader(input), &out)
	if err != nil {
		t.Fatal(err)
	}
	if err := ui.Run(); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestConsoleScenarioPersistsAndExportsCalculatedResults(t *testing.T) {
	dir := t.TempDir()
	data, destination := filepath.Join(dir, "course.json"), filepath.Join(dir, "results.csv")
	input := strings.Join([]string{
		"1", "s1", "Анна; \"Иванова\"",
		"1", "s2", "Борис",
		"5", "hw", "ДЗ", "20", "40",
		"5", "exam", "Экзамен", "10", "60",
		"9", "s1", "hw", "16", "Комментарий",
		"9", "s1", "exam", "9", "",
		"9", "s2", "hw", "10", "",
		"9", "s1", "exam", "12", "Недопустимый балл",
		"9", "s1", "absent", "1", "Нет критерия",
		"12", "s1", "13",
		"15", "strict", "12", "s2", "15", "zero",
		"2", "6", "10", "s1", "14", "", "0", "",
	}, "\n")
	output := runUI(t, data, destination, input)
	for _, expected := range []string{"8.60 / 10", "2.00 / 10", "от 0 до 10.00", "критерий absent", "выставлены не все оценки", "средний итог: 5.30", "Итоги экспортированы"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("missing %q in output:\n%s", expected, output)
		}
	}
	content, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	reader := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(content, []byte{0xef, 0xbb, 0xbf})))
	reader.Comma = ';'
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[1][1] != "Анна; \"Иванова\"" || rows[1][2] != "8.60" || rows[2][2] != "2.00" {
		t.Fatalf("unexpected CSV: %#v", rows)
	}

	restarted := runUI(t, data, destination, "12\ns1\n0\n")
	if !strings.Contains(restarted, "8.60 / 10") {
		t.Fatalf("results were not restored:\n%s", restarted)
	}
	before, err := os.ReadFile(data)
	if err != nil {
		t.Fatal(err)
	}
	protected := runUI(t, data, destination, "14\n"+data+"\n0\n")
	if !strings.Contains(protected, "нельзя экспортировать поверх файла хранилища") {
		t.Fatalf("data overwrite was allowed: %s", protected)
	}
	after, err := os.ReadFile(data)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("attempted export altered the data file", err)
	}
}

func TestInvalidInputAndEOFDoNotCreatePartialRecords(t *testing.T) {
	dir := t.TempDir()
	data := filepath.Join(dir, "course.json")
	out := runUI(t, data, filepath.Join(dir, "r.csv"), "oops\n5\nc1\nКритерий\nNaN\n1\n1\ns1\n")
	if !strings.Contains(out, "неизвестная команда") || !strings.Contains(out, "некорректные данные") {
		t.Fatalf("input errors were not explained: %s", out)
	}
	if _, err := os.Stat(data); !os.IsNotExist(err) {
		t.Fatalf("incomplete command wrote data: %v", err)
	}
}

func TestDemoAndCorruptedStorageAreHandledWithoutOverwriting(t *testing.T) {
	dir := t.TempDir()
	data, destination := filepath.Join(dir, "course.json"), filepath.Join(dir, "r.csv")
	var output bytes.Buffer
	ui, err := compose(data, destination, strings.NewReader(""), &output)
	if err != nil {
		t.Fatal(err)
	}
	if err := ui.Demo(destination); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := ui.Demo(destination); err == nil {
		t.Fatal("demo accepted nonempty user data")
	}
	after, err := os.ReadFile(data)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("refused demo changed data", err)
	}

	var document map[string]any
	if err := json.Unmarshal(before, &document); err != nil {
		t.Fatal(err)
	}
	evaluations := document["evaluations"].([]any)
	evaluations[0].(map[string]any)["Score"] = "999"
	forged, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(data, forged, 0600); err != nil {
		t.Fatal(err)
	}
	if err := ui.Run(); err == nil {
		t.Fatal("corrupt storage was accepted")
	}
	untouched, err := os.ReadFile(data)
	if err != nil || !bytes.Equal(forged, untouched) {
		t.Fatal("corrupt storage was overwritten", err)
	}
}

func TestConsoleCorrectionCancellationAndDeletion(t *testing.T) {
	dir := t.TempDir()
	data, destination := filepath.Join(dir, "course.json"), filepath.Join(dir, "r.csv")
	var output bytes.Buffer
	ui, err := compose(data, destination, strings.NewReader(""), &output)
	if err != nil {
		t.Fatal(err)
	}
	if err := ui.Demo(destination); err != nil {
		t.Fatal(err)
	}
	script := strings.Join([]string{
		"3", "s1", "Анна Новая",
		"7", "hw", "ДЗ-2", "10", "40", 
		"9", "s1", "hw", "8", "Исправлено",
		"7", "hw", "ДЗ-2", "10", "40", 
		"12", "s1", "10", "s1",
		"8", "hw", 
		"11", "s1", "exam", "8", "exam",
		"4", "s2", "11", "s1", "hw", "8", "hw",
		"4", "s1", "2", "6", "0", "",
	}, "\n")
	got := runUI(t, data, destination, script)
	for _, text := range []string{"новая граница ниже", "Анна Новая | 8.60", "Исправлено", "сначала отмените оценки", "Список пуст.", "Рубрикатор пуст."} {
		if !strings.Contains(got, text) {
			t.Fatalf("missing %q in output:\n%s", text, got)
		}
	}
	content, err := os.ReadFile(data)
	if err != nil {
		t.Fatal(err)
	}
	var stored struct {
		Students, Criteria, Evaluations []any
	}
	if err := json.Unmarshal(content, &stored); err != nil {
		t.Fatal(err)
	}
	if len(stored.Students)+len(stored.Criteria)+len(stored.Evaluations) != 0 {
		t.Fatalf("deleted data persisted: %s", content)
	}
}
