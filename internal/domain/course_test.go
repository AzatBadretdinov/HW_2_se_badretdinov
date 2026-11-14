package domain_test

import (
	"errors"
	"testing"

	"hw2kpo/internal/domain"
)

func id(t *testing.T, value string) domain.ID {
	t.Helper()
	v, err := domain.NewID(value)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func score(t *testing.T, text string) domain.Score {
	t.Helper()
	v, err := domain.ParseScore(text)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func weight(t *testing.T, text string) domain.Weight {
	t.Helper()
	v, err := domain.ParseWeight(text)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T) *domain.Course {
	t.Helper()
	c := domain.NewCourse()
	must(t, c.AddStudent(id(t, "s1"), "Анна Иванова"))
	must(t, c.AddCriterion(id(t, "hw"), "Домашняя работа", score(t, "20"), weight(t, "40")))
	must(t, c.AddCriterion(id(t, "exam"), "Экзамен", score(t, "10"), weight(t, "60")))
	return c
}

func TestScoreInputIsExactAndRejectsNonFiniteOrOverPreciseValues(t *testing.T) {
	for _, text := range []string{"-1", "NaN", "Inf", "1e2", "1.001", "1000000.01", "", ".5", "1.", "1,2.3"} {
		t.Run(text, func(t *testing.T) {
			if _, err := domain.ParseScore(text); !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("input %q: expected invalid, got %v", text, err)
			}
		})
	}
	for text, expected := range map[string]string{"0": "0.00", "12,3": "12.30", "0.01": "0.01", "1000000": "1000000.00"} {
		if got := score(t, text).String(); got != expected {
			t.Fatalf("%s: got %s, want %s", text, got, expected)
		}
	}
	if _, err := domain.ParseWeight("0"); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("zero weight: %v", err)
	}
}

func TestEvaluationRequiresExistingStudentCriterionAndBounds(t *testing.T) {
	c := fixture(t)
	s, cr := id(t, "s1"), id(t, "exam")
	for _, text := range []string{"0", "10"} {
		must(t, c.SetEvaluation(s, cr, score(t, text), ""))
	}
	if err := c.SetEvaluation(s, cr, score(t, "10.01"), ""); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("overflow accepted: %v", err)
	}
	if e, ok := c.Evaluation(s, cr); !ok || e.Score().String() != "10.00" {
		t.Fatal("failed correction changed the existing evaluation")
	}
	if err := c.SetEvaluation(s, id(t, "missing"), score(t, "1"), ""); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown criterion accepted: %v", err)
	}
	if err := c.SetEvaluation(id(t, "missing"), cr, score(t, "1"), ""); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown student accepted: %v", err)
	}
}

func TestFinalTracksCorrectionsCancellationAndCurrentRubric(t *testing.T) {
	c := fixture(t)
	s, hw, exam := id(t, "s1"), id(t, "hw"), id(t, "exam")
	calculator := domain.FinalScoreService{}
	check := func(expected float64, assessed, total int) {
		t.Helper()
		got, err := calculator.Calculate(c, s, domain.WeightedZero{})
		must(t, err)
		if got.Value != expected || got.Assessed != assessed || got.Total != total {
			t.Fatalf("got %+v, want %.2f (%d/%d)", got, expected, assessed, total)
		}
	}
	check(0, 0, 2)
	must(t, c.SetEvaluation(s, hw, score(t, "16"), ""))
	check(3.2, 1, 2)
	if _, err := calculator.Calculate(c, s, domain.WeightedComplete{}); !errors.Is(err, domain.ErrIncomplete) {
		t.Fatalf("strict accepted missing score: %v", err)
	}
	must(t, c.SetEvaluation(s, exam, score(t, "9"), ""))
	check(8.6, 2, 2)
	strict, err := calculator.Calculate(c, s, domain.WeightedComplete{})
	must(t, err)
	if strict.Value != 8.6 {
		t.Fatalf("strict total: %+v", strict)
	}
	must(t, c.SetEvaluation(s, exam, score(t, "5"), "исправлено"))
	check(6.2, 2, 2)
	must(t, c.CancelEvaluation(s, exam))
	check(3.2, 1, 2)
	must(t, c.UpdateCriterion(hw, "ДЗ", score(t, "20"), weight(t, "60")))
	check(4, 1, 2)
	must(t, c.AddCriterion(id(t, "project"), "Проект", score(t, "100"), weight(t, "80")))
	check(2.4, 1, 3)
}

func TestCriterionChangesCannotInvalidateExistingEvaluations(t *testing.T) {
	c := fixture(t)
	s, cr := id(t, "s1"), id(t, "exam")
	must(t, c.SetEvaluation(s, cr, score(t, "9"), ""))
	if err := c.UpdateCriterion(cr, "Новое имя", score(t, "8"), weight(t, "1")); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("unsafe maximum change: %v", err)
	}
	current, err := c.Criterion(cr)
	must(t, err)
	if current.Name() != "Экзамен" || current.Max().String() != "10.00" || current.Weight().String() != "60.00" {
		t.Fatal("rejected update was partially applied")
	}
	if err := c.RemoveCriterion(cr); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("criterion with evaluations removed: %v", err)
	}
	must(t, c.CancelEvaluation(s, cr))
	must(t, c.RemoveCriterion(cr))
	if err := c.SetEvaluation(s, cr, score(t, "1"), ""); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("removed criterion accepted: %v", err)
	}
}

func TestAggregateProtectsIdentityAndCollections(t *testing.T) {
	c := fixture(t)
	s := id(t, "s1")
	if err := c.AddStudent(s, "Другой студент"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate ID accepted: %v", err)
	}
	if err := c.RenameStudent(s, " \t "); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("empty name accepted: %v", err)
	}
	if err := c.AddCriterion(id(t, "bad"), "Критерий", domain.Score{}, weight(t, "1")); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("zero maximum accepted: %v", err)
	}
	must(t, c.RenameStudent(s, "Новое имя"))
	students := c.Students()
	students[0] = domain.Student{}
	stored, err := c.Student(s)
	must(t, err)
	if stored.Name() != "Новое имя" {
		t.Fatal("collection leaked mutable state")
	}
	must(t, c.SetEvaluation(s, id(t, "hw"), score(t, "10"), ""))
	must(t, c.RemoveStudent(s))
	if _, exists := c.Evaluation(s, id(t, "hw")); exists {
		t.Fatal("student deletion left an orphan evaluation")
	}
	// Reusing an ID must not resurrect the previous student's evaluations.
	must(t, c.AddStudent(s, "Новый студент"))
	result, err := (domain.FinalScoreService{}).Calculate(c, s, domain.WeightedZero{})
	must(t, err)
	if result.Value != 0 || result.Assessed != 0 {
		t.Fatalf("orphan score resurrected: %+v", result)
	}
}

func TestEmptyRubricAndUnknownStudentHaveNoFinalScore(t *testing.T) {
	c := domain.NewCourse()
	s := id(t, "s1")
	calculator := domain.FinalScoreService{}
	if _, err := calculator.Calculate(c, s, domain.WeightedZero{}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown student: %v", err)
	}
	must(t, c.AddStudent(s, "Анна"))
	if _, err := calculator.Calculate(c, s, domain.WeightedZero{}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("empty rubric: %v", err)
	}
}

func TestBuilderResolvesReferencesAndReturnsOnlyIndependentValidAggregates(t *testing.T) {
	builder := domain.NewCourseBuilder().
		WithEvaluation("s1", "c1", "7.50", "Комментарий").
		WithCriterion("c1", "ДЗ", "10", "1").
		WithStudent("s1", "Анна")
	first, err := builder.Build()
	must(t, err)
	second, err := builder.Build()
	must(t, err)
	must(t, first.RemoveStudent(id(t, "s1")))
	if _, err := second.Student(id(t, "s1")); err != nil {
		t.Fatal("builder products share mutable state", err)
	}
	result, err := (domain.FinalScoreService{}).Calculate(second, id(t, "s1"), domain.WeightedComplete{})
	must(t, err)
	if result.Value != 7.5 {
		t.Fatalf("builder lost evaluation: %+v", result)
	}
	broken, err := builder.WithEvaluation("s1", "missing", "1", "").Build()
	if !errors.Is(err, domain.ErrNotFound) || broken != nil {
		t.Fatalf("builder exposed an invalid partial product: %v, %v", broken, err)
	}
}
