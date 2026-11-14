package console

import "fmt"


func (u *UI) Demo(exportPath string) error {
	students, err := u.app.Students()
	if err != nil {
		return err
	}
	criteria, err := u.app.Criteria()
	if err != nil {
		return err
	}
	if len(students) != 0 || len(criteria) != 0 {
		return fmt.Errorf("демонстрации нужно пустое хранилище; укажите новый путь через --data")
	}
	steps := []struct {
		label string
		run   func() error
	}{
		{"Добавляем студента s1: Анна Иванова", func() error { return u.app.AddStudent("s1", "Анна Иванова") }},
		{"Добавляем студента s2: Борис Петров", func() error { return u.app.AddStudent("s2", "Борис Петров") }},
		{"Критерий hw: Домашняя работа, максимум 20, вес 40", func() error { return u.app.AddCriterion("hw", "Домашняя работа", "20", "40") }},
		{"Критерий exam: Экзамен, максимум 10, вес 60", func() error { return u.app.AddCriterion("exam", "Экзамен", "10", "60") }},
		{"Оцениваем s1: hw = 16/20", func() error { return u.app.SetEvaluation("s1", "hw", "16", "Хорошая работа") }},
		{"Оцениваем s1: exam = 9/10", func() error { return u.app.SetEvaluation("s1", "exam", "9", "") }},
		{"Оцениваем s2: hw = 10/20", func() error { return u.app.SetEvaluation("s2", "hw", "10", "") }},
	}
	for _, step := range steps {
		fmt.Fprintln(u.output, step.label)
		if err := step.run(); err != nil {
			return err
		}
	}
	fmt.Fprintln(u.output, "Пробуем поставить 12 баллов из 10:")
	if err := u.app.SetEvaluation("s1", "exam", "12", ""); err != nil {
		fmt.Fprintln(u.output, "Ожидаемый отказ:", err)
	} else {
		return fmt.Errorf("ошибка демонстрации: недопустимая оценка принята")
	}
	fmt.Fprintln(u.output, "Проверяем strict для s2 с пропущенным экзаменом:")
	if _, err := u.app.Final("s2", "strict"); err != nil {
		fmt.Fprintln(u.output, "Ожидаемый отказ:", err)
	} else {
		return fmt.Errorf("ошибка демонстрации: принят неполный набор оценок")
	}
	fmt.Fprintln(u.output, "Ведомость zero: ожидаем s1 = 8.60, s2 = 2.00")
	rows, err := u.app.Results("zero")
	if err != nil {
		return err
	}
	u.printResults(rows)
	if err := u.app.Export("zero", exportPath); err != nil {
		return err
	}
	fmt.Fprintln(u.output, "Экспорт:", exportPath)
	return nil
}
