package console

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"hw2kpo/internal/application"
)

const menu = `
1 Добавить студента       2 Список студентов
3 Переименовать студента  4 Удалить студента и его оценки
5 Добавить критерий      6 Список критериев
7 Изменить критерий      8 Удалить критерий без оценок
9 Выставить/исправить оценку
10 Оценки студента       11 Отменить оценку
12 Итог студента         13 Ведомость и статистика
14 Экспорт CSV           15 Режим расчёта (zero / strict)
0 Выход
`

type UI struct {
	app           *application.Facade
	input         *bufio.Scanner
	output        io.Writer
	policy        string
	defaultExport string
}

func New(app *application.Facade, input io.Reader, output io.Writer, defaultExport string) *UI {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	return &UI{app, scanner, output, "zero", defaultExport}
}

func (u *UI) read(prompt string) (string, error) {
	fmt.Fprint(u.output, prompt+": ")
	if !u.input.Scan() {
		if err := u.input.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return strings.TrimSpace(u.input.Text()), nil
}

func (u *UI) fields(prompts ...string) ([]string, error) {
	values := make([]string, 0, len(prompts))
	for _, prompt := range prompts {
		value, err := u.read(prompt)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (u *UI) Run() error {

	if _, err := u.app.Students(); err != nil {
		return err
	}
	fmt.Fprintln(u.output, "ПОС — преподавательская оценка студентов")
	fmt.Fprintln(u.output, "Числа: до 2 знаков после точки/запятой. Пропуски в режиме zero дают 0.")
	for {
		fmt.Fprint(u.output, menu)
		command, err := u.read("Команда [режим " + u.policy + "]")
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if command == "0" {
			fmt.Fprintln(u.output, "Данные сохранены. До свидания!")
			return nil
		}
		if err := u.execute(command); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			fmt.Fprintln(u.output, "Ошибка:", err)
		}
	}
}

func (u *UI) execute(command string) error {
	switch command {
	case "1", "3":
		v, err := u.fields("ID студента", "ФИО")
		if err != nil {
			return err
		}
		if command == "1" {
			err = u.app.AddStudent(v[0], v[1])
		} else {
			err = u.app.RenameStudent(v[0], v[1])
		}
		return u.saved(err)
	case "2":
		students, err := u.app.Students()
		if err != nil {
			return err
		}
		fmt.Fprintln(u.output, "Студенты (ID | ФИО):")
		for _, s := range students {
			fmt.Fprintf(u.output, "%s | %s\n", s.ID, s.Name)
		}
		if len(students) == 0 {
			fmt.Fprintln(u.output, "Список пуст.")
		}
	case "4":
		id, err := u.read("ID удаляемого студента (его оценки тоже будут удалены)")
		if err != nil {
			return err
		}
		return u.saved(u.app.RemoveStudent(id))
	case "5", "7":
		v, err := u.fields("ID критерия", "Название", "Максимальный балл (> 0)", "Вес (> 0)")
		if err != nil {
			return err
		}
		if command == "5" {
			err = u.app.AddCriterion(v[0], v[1], v[2], v[3])
		} else {
			err = u.app.UpdateCriterion(v[0], v[1], v[2], v[3])
		}
		return u.saved(err)
	case "6":
		criteria, err := u.app.Criteria()
		if err != nil {
			return err
		}
		fmt.Fprintln(u.output, "Критерии (ID | Название | Максимум | Вес):")
		for _, cr := range criteria {
			fmt.Fprintf(u.output, "%s | %s | %s | %s\n", cr.ID, cr.Name, cr.Max, cr.Weight)
		}
		if len(criteria) == 0 {
			fmt.Fprintln(u.output, "Рубрикатор пуст.")
		}
	case "8":
		id, err := u.read("ID удаляемого критерия")
		if err != nil {
			return err
		}
		return u.saved(u.app.RemoveCriterion(id))
	case "9":
		v, err := u.fields("ID студента", "ID критерия", "Балл", "Комментарий (можно пустой)")
		if err != nil {
			return err
		}
		return u.saved(u.app.SetEvaluation(v[0], v[1], v[2], v[3]))
	case "10":
		id, err := u.read("ID студента")
		if err != nil {
			return err
		}
		rows, err := u.app.Evaluations(id)
		if err != nil {
			return err
		}
		fmt.Fprintln(u.output, "Оценки (Критерий | Название | Балл / Максимум | Комментарий):")
		for _, row := range rows {
			fmt.Fprintf(u.output, "%s | %s | %s / %s | %s\n", row.CriterionID, row.CriterionName, row.Score, row.Max, row.Comment)
		}
		if len(rows) == 0 {
			fmt.Fprintln(u.output, "Рубрикатор пуст.")
		}
	case "11":
		v, err := u.fields("ID студента", "ID критерия")
		if err != nil {
			return err
		}
		return u.saved(u.app.CancelEvaluation(v[0], v[1]))
	case "12":
		id, err := u.read("ID студента")
		if err != nil {
			return err
		}
		row, err := u.app.Final(id, u.policy)
		if err != nil {
			return err
		}
		u.printResult(row)
	case "13":
		rows, err := u.app.Results(u.policy)
		if err != nil {
			return err
		}
		u.printResults(rows)
	case "14":
		path, err := u.read("Файл CSV (Enter = " + u.defaultExport + ")")
		if err != nil {
			return err
		}
		if path == "" {
			path = u.defaultExport
		}
		if err := u.app.Export(u.policy, path); err != nil {
			return err
		}
		fmt.Fprintln(u.output, "Итоги экспортированы:", path)
	case "15":
		name, err := u.read("Режим: zero (пропуски = 0), strict (нужны все оценки)")
		if err != nil {
			return err
		}
		if err := u.app.CheckPolicy(name); err != nil {
			return err
		}
		u.policy = name
		fmt.Fprintln(u.output, "Режим расчёта:", name)
	default:
		return fmt.Errorf("неизвестная команда %q; выберите номер из меню", command)
	}
	return nil
}

func (u *UI) saved(err error) error {
	if err == nil {
		fmt.Fprintln(u.output, "Сохранено.")
	}
	return err
}

func (u *UI) printResult(row application.ResultRow) {
	fmt.Fprintf(u.output, "%s | %s | %.2f / 10 | оценено %d/%d | %s\n", row.StudentID, row.StudentName, row.Score, row.Assessed, row.Total, row.Policy)
}

func (u *UI) printResults(rows []application.ResultRow) {
	if len(rows) == 0 {
		fmt.Fprintln(u.output, "Студентов пока нет.")
		return
	}
	min, max, sum := rows[0].Score, rows[0].Score, 0.0
	for _, row := range rows {
		u.printResult(row)
		sum += row.Score
		if row.Score < min {
			min = row.Score
		}
		if row.Score > max {
			max = row.Score
		}
	}
	fmt.Fprintf(u.output, "Студентов: %d; средний итог: %.2f; минимум: %.2f; максимум: %.2f\n", len(rows), sum/float64(len(rows)), min, max)
}
