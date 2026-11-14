package domain

import "fmt"

type studentSpec struct{ id, name string }
type criterionSpec struct{ id, name, maximum, weight string }
type evaluationSpec struct{ student, criterion, score, comment string }


type CourseBuilder struct {
	students    []studentSpec
	criteria    []criterionSpec
	evaluations []evaluationSpec
}

func NewCourseBuilder() *CourseBuilder { return &CourseBuilder{} }

func (b *CourseBuilder) WithStudent(id, name string) *CourseBuilder {
	b.students = append(b.students, studentSpec{id, name})
	return b
}

func (b *CourseBuilder) WithCriterion(id, name, maximum, weight string) *CourseBuilder {
	b.criteria = append(b.criteria, criterionSpec{id, name, maximum, weight})
	return b
}

func (b *CourseBuilder) WithEvaluation(student, criterion, score, comment string) *CourseBuilder {
	b.evaluations = append(b.evaluations, evaluationSpec{student, criterion, score, comment})
	return b
}

func (b *CourseBuilder) Build() (*Course, error) {
	c := NewCourse()
	for _, s := range b.students {
		id, err := NewID(s.id)
		if err != nil {
			return nil, err
		}
		if err := c.AddStudent(id, s.name); err != nil {
			return nil, err
		}
	}
	for _, cr := range b.criteria {
		id, err := NewID(cr.id)
		if err != nil {
			return nil, err
		}
		max, err := ParseScore(cr.maximum)
		if err != nil {
			return nil, err
		}
		weight, err := ParseWeight(cr.weight)
		if err != nil {
			return nil, err
		}
		if err := c.AddCriterion(id, cr.name, max, weight); err != nil {
			return nil, err
		}
	}
	for _, e := range b.evaluations {
		s, err := NewID(e.student)
		if err != nil {
			return nil, err
		}
		cr, err := NewID(e.criterion)
		if err != nil {
			return nil, err
		}
		score, err := ParseScore(e.score)
		if err != nil {
			return nil, err
		}
		if _, exists := c.Evaluation(s, cr); exists {
			return nil, fmt.Errorf("%w: дублирующая оценка %s / %s", ErrConflict, s, cr)
		}
		if err := c.SetEvaluation(s, cr, score, e.comment); err != nil {
			return nil, err
		}
	}
	return c, nil
}
