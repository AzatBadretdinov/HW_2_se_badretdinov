package storage

import (
	"strings"
	"testing"
)

const validDocument = `{
  "version": 1,
  "students": [{"ID":"s1","Name":"Анна"}],
  "criteria": [{"ID":"c1","Name":"ДЗ","MaxScore":"10.00","Weight":"1.00"}],
  "evaluations": [{"StudentID":"s1","CriterionID":"c1","Score":"8.50","Comment":"Отзыв\nв две строки"}]
}`

func TestStorageRestorationRejectsInvalidSnapshots(t *testing.T) {
	if _, err := decode([]byte(validDocument)); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"empty file":         "",
		"unknown version":    strings.Replace(validDocument, `"version": 1`, `"version": 2`, 1),
		"unknown field":      strings.Replace(validDocument, `"version": 1`, `"extra":true,"version":1`, 1),
		"unknown student":    strings.Replace(validDocument, `"StudentID":"s1"`, `"StudentID":"s2"`, 1),
		"unknown criterion":  strings.Replace(validDocument, `"CriterionID":"c1"`, `"CriterionID":"c2"`, 1),
		"score out of range": strings.Replace(validDocument, `"Score":"8.50"`, `"Score":"10.01"`, 1),
		"duplicate student":  strings.Replace(validDocument, `[{"ID":"s1","Name":"Анна"}]`, `[{"ID":"s1","Name":"Анна"},{"ID":"s1","Name":"Борис"}]`, 1),
		"duplicate score": strings.Replace(validDocument,
			`[{"StudentID":"s1","CriterionID":"c1","Score":"8.50","Comment":"Отзыв\nв две строки"}]`,
			`[{"StudentID":"s1","CriterionID":"c1","Score":"8.50"},{"StudentID":"s1","CriterionID":"c1","Score":"7.00"}]`, 1),
		"two JSON documents": validDocument + "{}",
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := decode([]byte(input)); err == nil {
				t.Fatal("invalid snapshot was accepted")
			}
		})
	}
}
