package domain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	ErrInvalid    = errors.New("некорректные данные")
	ErrNotFound   = errors.New("объект не найден")
	ErrConflict   = errors.New("конфликт данных")
	ErrIncomplete = errors.New("выставлены не все оценки")
)


type ID struct{ value string }

func NewID(value string) (ID, error) {
	value = strings.TrimSpace(value)
	if value == "" || utf8.RuneCountInString(value) > 64 {
		return ID{}, fmt.Errorf("%w: ID должен содержать от 1 до 64 символов", ErrInvalid)
	}
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("._-", r) {
			return ID{}, fmt.Errorf("%w: ID может содержать буквы, цифры, точку, дефис и подчёркивание", ErrInvalid)
		}
	}
	return ID{value}, nil
}

func (id ID) String() string { return id.value }


const maxHundredths int64 = 100_000_000

type Score struct{ hundredths int64 }
type Weight struct{ hundredths int64 }

func ParseScore(text string) (Score, error) {
	n, err := parseHundredths(text)
	if err != nil {
		return Score{}, err
	}
	return Score{n}, nil
}

func ParseWeight(text string) (Weight, error) {
	n, err := parseHundredths(text)
	if err != nil {
		return Weight{}, err
	}
	if n == 0 {
		return Weight{}, fmt.Errorf("%w: вес должен быть положительным", ErrInvalid)
	}
	return Weight{n}, nil
}

func (s Score) String() string  { return formatHundredths(s.hundredths) }
func (w Weight) String() string { return formatHundredths(w.hundredths) }

func parseHundredths(text string) (int64, error) {
	text = strings.ReplaceAll(strings.TrimSpace(text), ",", ".")
	invalid := func() (int64, error) {
		return 0, fmt.Errorf("%w: нужно число от 0 до 1000000, не более двух знаков после запятой", ErrInvalid)
	}
	parts := strings.Split(text, ".")
	if len(parts) > 2 || parts[0] == "" || len(parts[0]) > 7 {
		return invalid()
	}
	for _, p := range parts {
		if p == "" {
			return invalid()
		}
		for _, r := range p {
			if r < '0' || r > '9' {
				return invalid()
			}
		}
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return invalid()
	}
	var fraction int64
	if len(parts) == 2 {
		if len(parts[1]) > 2 {
			return invalid()
		}
		fraction, _ = strconv.ParseInt(parts[1]+strings.Repeat("0", 2-len(parts[1])), 10, 64)
	}
	n := whole*100 + fraction
	if n > maxHundredths {
		return invalid()
	}
	return n, nil
}

func formatHundredths(n int64) string { return fmt.Sprintf("%d.%02d", n/100, n%100) }

func validName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 200 {
		return "", fmt.Errorf("%w: имя/название должно содержать от 1 до 200 символов", ErrInvalid)
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", fmt.Errorf("%w: имя/название не может содержать управляющие символы", ErrInvalid)
		}
	}
	return name, nil
}
