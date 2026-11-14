package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"hw2kpo/internal/adapters/files"
	"hw2kpo/internal/domain"
)

type studentDTO struct{ ID, Name string }
type criterionDTO struct{ ID, Name, MaxScore, Weight string }
type evaluationDTO struct{ StudentID, CriterionID, Score, Comment string }
type courseDTO struct {
	Version     int             `json:"version"`
	Students    []studentDTO    `json:"students"`
	Criteria    []criterionDTO  `json:"criteria"`
	Evaluations []evaluationDTO `json:"evaluations"`
}

type JSONRepository struct{ path string }

func NewJSONRepository(path string) *JSONRepository { return &JSONRepository{path: path} }

func (r *JSONRepository) Load() (*domain.Course, error) {
	data, err := os.ReadFile(r.path)
	if errors.Is(err, os.ErrNotExist) {
		return domain.NewCourse(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("чтение хранилища: %w", err)
	}
	c, err := decode(data)
	if err != nil {
		return nil, fmt.Errorf("хранилище %s некорректно (исходный файл сохранён): %w", r.path, err)
	}
	return c, nil
}

func decode(data []byte) (*domain.Course, error) {
	var dto courseDTO
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&dto); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("после JSON есть лишние данные")
	}
	if dto.Version != 1 {
		return nil, fmt.Errorf("неподдерживаемая версия %d", dto.Version)
	}
	builder := domain.NewCourseBuilder()
	for _, s := range dto.Students {
		builder.WithStudent(s.ID, s.Name)
	}
	for _, cr := range dto.Criteria {
		builder.WithCriterion(cr.ID, cr.Name, cr.MaxScore, cr.Weight)
	}
	for _, e := range dto.Evaluations {
		builder.WithEvaluation(e.StudentID, e.CriterionID, e.Score, e.Comment)
	}
	return builder.Build()
}

func (r *JSONRepository) Save(c *domain.Course) error {
	dto := courseDTO{Version: 1, Students: []studentDTO{}, Criteria: []criterionDTO{}, Evaluations: []evaluationDTO{}}
	for _, s := range c.Students() {
		dto.Students = append(dto.Students, studentDTO{s.ID().String(), s.Name()})
		for _, cr := range c.Criteria() {
			if e, ok := c.Evaluation(s.ID(), cr.ID()); ok {
				dto.Evaluations = append(dto.Evaluations, evaluationDTO{s.ID().String(), cr.ID().String(), e.Score().String(), e.Comment()})
			}
		}
	}
	for _, cr := range c.Criteria() {
		dto.Criteria = append(dto.Criteria, criterionDTO{cr.ID().String(), cr.Name(), cr.Max().String(), cr.Weight().String()})
	}
	return files.Write(r.path, func(w io.Writer) error {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(dto)
	})
}

var _ domain.CourseRepository = (*JSONRepository)(nil)
