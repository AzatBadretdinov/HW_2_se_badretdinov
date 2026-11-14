package application

import (
	"fmt"

	"hw2kpo/internal/domain"
)

type StudentView struct{ ID, Name string }
type CriterionView struct{ ID, Name, Max, Weight string }
type EvaluationView struct {
	CriterionID, CriterionName, Score, Max, Comment string
	Present                                         bool
}
type ResultRow struct {
	StudentID, StudentName string
	Score                  float64
	Assessed, Total        int
	Policy                 string
}


type ResultsExporter interface {
	Export(rows []ResultRow, destination string) error
}


type Policies map[string]domain.ScoreStrategy


type Facade struct {
	repository domain.CourseRepository
	calculator domain.FinalScoreService
	exporter   ResultsExporter
	policies   Policies
}

func NewFacade(repo domain.CourseRepository, calculator domain.FinalScoreService, exporter ResultsExporter, policies Policies) *Facade {
	copyPolicies := make(Policies, len(policies))
	for name, strategy := range policies {
		copyPolicies[name] = strategy
	}
	return &Facade{repo, calculator, exporter, copyPolicies}
}

func (f *Facade) mutate(action func(*domain.Course) error) error {
	c, err := f.repository.Load()
	if err != nil {
		return err
	}
	if err := action(c); err != nil {
		return err
	}
	return f.repository.Save(c)
}

func (f *Facade) AddStudent(id, name string) error {
	student, err := domain.NewID(id)
	if err != nil {
		return err
	}
	return f.mutate(func(c *domain.Course) error { return c.AddStudent(student, name) })
}

func (f *Facade) RenameStudent(id, name string) error {
	student, err := domain.NewID(id)
	if err != nil {
		return err
	}
	return f.mutate(func(c *domain.Course) error { return c.RenameStudent(student, name) })
}

func (f *Facade) RemoveStudent(id string) error {
	student, err := domain.NewID(id)
	if err != nil {
		return err
	}
	return f.mutate(func(c *domain.Course) error { return c.RemoveStudent(student) })
}

func (f *Facade) Students() ([]StudentView, error) {
	c, err := f.repository.Load()
	if err != nil {
		return nil, err
	}
	rows := make([]StudentView, 0)
	for _, s := range c.Students() {
		rows = append(rows, StudentView{s.ID().String(), s.Name()})
	}
	return rows, nil
}

func (f *Facade) saveCriterion(id, name, maximum, weight string, update bool) error {
	criterion, err := domain.NewID(id)
	if err != nil {
		return err
	}
	max, err := domain.ParseScore(maximum)
	if err != nil {
		return err
	}
	w, err := domain.ParseWeight(weight)
	if err != nil {
		return err
	}
	return f.mutate(func(c *domain.Course) error {
		if update {
			return c.UpdateCriterion(criterion, name, max, w)
		}
		return c.AddCriterion(criterion, name, max, w)
	})
}

func (f *Facade) AddCriterion(id, name, max, weight string) error {
	return f.saveCriterion(id, name, max, weight, false)
}

func (f *Facade) UpdateCriterion(id, name, max, weight string) error {
	return f.saveCriterion(id, name, max, weight, true)
}

func (f *Facade) RemoveCriterion(id string) error {
	criterion, err := domain.NewID(id)
	if err != nil {
		return err
	}
	return f.mutate(func(c *domain.Course) error { return c.RemoveCriterion(criterion) })
}

func (f *Facade) Criteria() ([]CriterionView, error) {
	c, err := f.repository.Load()
	if err != nil {
		return nil, err
	}
	rows := make([]CriterionView, 0)
	for _, cr := range c.Criteria() {
		rows = append(rows, CriterionView{cr.ID().String(), cr.Name(), cr.Max().String(), cr.Weight().String()})
	}
	return rows, nil
}

func parsePair(student, criterion string) (domain.ID, domain.ID, error) {
	s, err := domain.NewID(student)
	if err != nil {
		return domain.ID{}, domain.ID{}, err
	}
	c, err := domain.NewID(criterion)
	return s, c, err
}

func (f *Facade) SetEvaluation(student, criterion, score, comment string) error {
	s, cr, err := parsePair(student, criterion)
	if err != nil {
		return err
	}
	value, err := domain.ParseScore(score)
	if err != nil {
		return err
	}
	return f.mutate(func(c *domain.Course) error { return c.SetEvaluation(s, cr, value, comment) })
}

func (f *Facade) CancelEvaluation(student, criterion string) error {
	s, cr, err := parsePair(student, criterion)
	if err != nil {
		return err
	}
	return f.mutate(func(c *domain.Course) error { return c.CancelEvaluation(s, cr) })
}

func (f *Facade) Evaluations(student string) ([]EvaluationView, error) {
	id, err := domain.NewID(student)
	if err != nil {
		return nil, err
	}
	c, err := f.repository.Load()
	if err != nil {
		return nil, err
	}
	if _, err := c.Student(id); err != nil {
		return nil, err
	}
	rows := make([]EvaluationView, 0)
	for _, cr := range c.Criteria() {
		row := EvaluationView{CriterionID: cr.ID().String(), CriterionName: cr.Name(), Max: cr.Max().String(), Score: "-"}
		if e, ok := c.Evaluation(id, cr.ID()); ok {
			row.Score, row.Comment, row.Present = e.Score().String(), e.Comment(), true
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (f *Facade) CheckPolicy(name string) error {
	if strategy, ok := f.policies[name]; !ok || strategy == nil {
		return fmt.Errorf("%w: режим %q не существует (доступны zero и strict)", domain.ErrInvalid, name)
	}
	return nil
}

func (f *Facade) result(c *domain.Course, s domain.Student, policy string) (ResultRow, error) {
	total, err := f.calculator.Calculate(c, s.ID(), f.policies[policy])
	if err != nil {
		return ResultRow{}, fmt.Errorf("студент %s: %w", s.ID(), err)
	}
	return ResultRow{s.ID().String(), s.Name(), total.Value, total.Assessed, total.Total, policy}, nil
}

func (f *Facade) Final(student, policy string) (ResultRow, error) {
	if err := f.CheckPolicy(policy); err != nil {
		return ResultRow{}, err
	}
	id, err := domain.NewID(student)
	if err != nil {
		return ResultRow{}, err
	}
	c, err := f.repository.Load()
	if err != nil {
		return ResultRow{}, err
	}
	s, err := c.Student(id)
	if err != nil {
		return ResultRow{}, err
	}
	return f.result(c, s, policy)
}


func (f *Facade) Results(policy string) ([]ResultRow, error) {
	if err := f.CheckPolicy(policy); err != nil {
		return nil, err
	}
	c, err := f.repository.Load()
	if err != nil {
		return nil, err
	}
	if len(c.Criteria()) == 0 {
		return nil, fmt.Errorf("%w: рубрикатор пуст", domain.ErrInvalid)
	}
	rows := make([]ResultRow, 0)
	for _, s := range c.Students() {
		row, err := f.result(c, s, policy)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (f *Facade) Export(policy, destination string) error {
	rows, err := f.Results(policy)
	if err != nil {
		return err
	}
	return f.exporter.Export(rows, destination)
}
