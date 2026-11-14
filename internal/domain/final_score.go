package domain

import (
	"fmt"
	"math"
)


type ScoreStrategy interface {
	Calculate(criteria []Criterion, evaluations []Evaluation) (float64, error)
}


type WeightedZero struct{}

func (WeightedZero) Calculate(criteria []Criterion, evaluations []Evaluation) (float64, error) {
	if len(criteria) == 0 {
		return 0, fmt.Errorf("%w: рубрикатор пуст", ErrInvalid)
	}
	byCriterion := make(map[ID]Score, len(evaluations))
	for _, e := range evaluations {
		byCriterion[e.criterion] = e.score
	}
	var weighted, totalWeight float64
	for _, cr := range criteria {
		weight := float64(cr.weight.hundredths)
		totalWeight += weight
		weighted += float64(byCriterion[cr.id].hundredths) / float64(cr.max.hundredths) * weight
	}
	return 10 * weighted / totalWeight, nil
}


type WeightedComplete struct{}

func (WeightedComplete) Calculate(criteria []Criterion, evaluations []Evaluation) (float64, error) {
	if len(evaluations) != len(criteria) {
		return 0, fmt.Errorf("%w: заполнено %d из %d критериев", ErrIncomplete, len(evaluations), len(criteria))
	}
	return (WeightedZero{}).Calculate(criteria, evaluations)
}


type FinalScore struct {
	Value    float64
	Assessed int
	Total    int
}


type FinalScoreService struct{}

func (FinalScoreService) Calculate(c *Course, student ID, strategy ScoreStrategy) (FinalScore, error) {
	if _, err := c.Student(student); err != nil {
		return FinalScore{}, err
	}
	criteria := c.Criteria()
	if len(criteria) == 0 {
		return FinalScore{}, fmt.Errorf("%w: сначала добавьте критерии", ErrInvalid)
	}
	evaluations := make([]Evaluation, 0, len(criteria))
	for _, cr := range criteria {
		if e, ok := c.Evaluation(student, cr.id); ok {
			evaluations = append(evaluations, e)
		}
	}
	if strategy == nil {
		return FinalScore{}, fmt.Errorf("%w: не выбрана стратегия расчёта", ErrInvalid)
	}
	value, err := strategy.Calculate(criteria, evaluations)
	if err != nil {
		return FinalScore{}, err
	}
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 10.000000001 {
		return FinalScore{}, fmt.Errorf("%w: стратегия вернула итог вне шкалы 0..10", ErrInvalid)
	}
	return FinalScore{math.Round(value*100) / 100, len(evaluations), len(criteria)}, nil
}
