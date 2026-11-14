package domain

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

type Student struct {
	id   ID
	name string
}

func (s Student) ID() ID       { return s.id }
func (s Student) Name() string { return s.name }

type Criterion struct {
	id     ID
	name   string
	max    Score
	weight Weight
}

func (c Criterion) ID() ID         { return c.id }
func (c Criterion) Name() string   { return c.name }
func (c Criterion) Max() Score     { return c.max }
func (c Criterion) Weight() Weight { return c.weight }


type Evaluation struct {
	student   ID
	criterion ID
	score     Score
	comment   string
}

func (e Evaluation) StudentID() ID   { return e.student }
func (e Evaluation) CriterionID() ID { return e.criterion }
func (e Evaluation) Score() Score    { return e.score }
func (e Evaluation) Comment() string { return e.comment }

type evaluationKey struct{ student, criterion ID }


type Course struct {
	students    map[ID]Student
	criteria    map[ID]Criterion
	evaluations map[evaluationKey]Evaluation
}

func NewCourse() *Course {
	return &Course{
		students: make(map[ID]Student), criteria: make(map[ID]Criterion),
		evaluations: make(map[evaluationKey]Evaluation),
	}
}

func (c *Course) AddStudent(id ID, name string) error {
	if _, ok := c.students[id]; ok {
		return fmt.Errorf("%w: студент %s уже существует", ErrConflict, id)
	}
	if id.value == "" {
		return fmt.Errorf("%w: пустой ID студента", ErrInvalid)
	}
	name, err := validName(name)
	if err != nil {
		return err
	}
	c.students[id] = Student{id, name}
	return nil
}

func (c *Course) RenameStudent(id ID, name string) error {
	if _, err := c.Student(id); err != nil {
		return err
	}
	name, err := validName(name)
	if err != nil {
		return err
	}
	c.students[id] = Student{id, name}
	return nil
}


func (c *Course) RemoveStudent(id ID) error {
	if _, err := c.Student(id); err != nil {
		return err
	}
	delete(c.students, id)
	for key := range c.evaluations {
		if key.student == id {
			delete(c.evaluations, key)
		}
	}
	return nil
}

func (c *Course) AddCriterion(id ID, name string, max Score, weight Weight) error {
	if _, ok := c.criteria[id]; ok {
		return fmt.Errorf("%w: критерий %s уже существует", ErrConflict, id)
	}
	criterion, err := makeCriterion(id, name, max, weight)
	if err != nil {
		return err
	}
	c.criteria[id] = criterion
	return nil
}

func makeCriterion(id ID, name string, max Score, weight Weight) (Criterion, error) {
	if id.value == "" || max.hundredths <= 0 || weight.hundredths <= 0 {
		return Criterion{}, fmt.Errorf("%w: у критерия нужны ID, положительные максимум и вес", ErrInvalid)
	}
	name, err := validName(name)
	if err != nil {
		return Criterion{}, err
	}
	return Criterion{id, name, max, weight}, nil
}

func (c *Course) UpdateCriterion(id ID, name string, max Score, weight Weight) error {
	if _, err := c.Criterion(id); err != nil {
		return err
	}
	criterion, err := makeCriterion(id, name, max, weight)
	if err != nil {
		return err
	}
	for _, e := range c.evaluations {
		if e.criterion == id && e.score.hundredths > max.hundredths {
			return fmt.Errorf("%w: новая граница ниже уже выставленной оценки студенту %s", ErrConflict, e.student)
		}
	}
	c.criteria[id] = criterion
	return nil
}

func (c *Course) RemoveCriterion(id ID) error {
	if _, err := c.Criterion(id); err != nil {
		return err
	}
	for _, e := range c.evaluations {
		if e.criterion == id {
			return fmt.Errorf("%w: сначала отмените оценки по критерию %s", ErrConflict, id)
		}
	}
	delete(c.criteria, id)
	return nil
}


type EvaluationFactory struct{}

func (EvaluationFactory) Create(c *Course, student, criterion ID, score Score, comment string) (Evaluation, error) {
	if _, err := c.Student(student); err != nil {
		return Evaluation{}, err
	}
	cr, err := c.Criterion(criterion)
	if err != nil {
		return Evaluation{}, err
	}
	if score.hundredths < 0 || score.hundredths > cr.max.hundredths {
		return Evaluation{}, fmt.Errorf("%w: балл должен быть от 0 до %s", ErrInvalid, cr.max)
	}
	comment = strings.TrimSpace(comment)
	if utf8.RuneCountInString(comment) > 1000 {
		return Evaluation{}, fmt.Errorf("%w: комментарий длиннее 1000 символов", ErrInvalid)
	}
	return Evaluation{student, criterion, score, comment}, nil
}

func (c *Course) SetEvaluation(student, criterion ID, score Score, comment string) error {
	e, err := (EvaluationFactory{}).Create(c, student, criterion, score, comment)
	if err != nil {
		return err
	}
	c.evaluations[evaluationKey{student, criterion}] = e
	return nil
}

func (c *Course) CancelEvaluation(student, criterion ID) error {
	if _, err := c.Student(student); err != nil {
		return err
	}
	if _, err := c.Criterion(criterion); err != nil {
		return err
	}
	key := evaluationKey{student, criterion}
	if _, ok := c.evaluations[key]; !ok {
		return fmt.Errorf("%w: оценка ещё не выставлена", ErrNotFound)
	}
	delete(c.evaluations, key)
	return nil
}

func (c *Course) Student(id ID) (Student, error) {
	s, ok := c.students[id]
	if !ok {
		return Student{}, fmt.Errorf("%w: студент %s", ErrNotFound, id)
	}
	return s, nil
}

func (c *Course) Criterion(id ID) (Criterion, error) {
	cr, ok := c.criteria[id]
	if !ok {
		return Criterion{}, fmt.Errorf("%w: критерий %s", ErrNotFound, id)
	}
	return cr, nil
}

func (c *Course) Evaluation(student, criterion ID) (Evaluation, bool) {
	e, ok := c.evaluations[evaluationKey{student, criterion}]
	return e, ok
}

func (c *Course) Students() []Student {
	result := make([]Student, 0, len(c.students))
	for _, s := range c.students {
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].id.value < result[j].id.value })
	return result
}

func (c *Course) Criteria() []Criterion {
	result := make([]Criterion, 0, len(c.criteria))
	for _, cr := range c.criteria {
		result = append(result, cr)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].id.value < result[j].id.value })
	return result
}

type CourseRepository interface {
	Load() (*Course, error)
	Save(*Course) error
}
