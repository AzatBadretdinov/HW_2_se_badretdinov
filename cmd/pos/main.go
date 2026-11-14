package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"hw2kpo/internal/adapters/console"
	"hw2kpo/internal/adapters/export"
	"hw2kpo/internal/adapters/storage"
	"hw2kpo/internal/application"
	"hw2kpo/internal/di"
	"hw2kpo/internal/domain"
)

func main() {
	data := flag.String("data", "", "путь JSON-хранилища (по умолчанию data/course.json)")
	destination := flag.String("export", "results.csv", "путь CSV по умолчанию")
	demo := flag.Bool("demo", false, "выполнить сквозную демонстрацию на пустом хранилище")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "Неизвестные аргументы:", flag.Args())
		os.Exit(2)
	}
	if *data == "" {
		*data = "data/course.json"
		if *demo {
			*data = "data/demo-course.json"
		}
	}
	ui, err := compose(*data, *destination, os.Stdin, os.Stdout)
	if err == nil {
		if *demo {
			err = ui.Demo(*destination)
		} else {
			err = ui.Run()
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка:", err)
		os.Exit(1)
	}
}


func compose(data, destination string, input io.Reader, output io.Writer) (*console.UI, error) {
	c := di.New()
	registrations := []error{
		di.Provide[domain.CourseRepository](c, func(*di.Container) (domain.CourseRepository, error) {
			return storage.NewJSONRepository(data), nil
		}),
		di.Provide[application.ResultsExporter](c, func(*di.Container) (application.ResultsExporter, error) {
			return export.NewCSVExporter(data), nil
		}),
		di.Provide[domain.FinalScoreService](c, func(*di.Container) (domain.FinalScoreService, error) {
			return domain.FinalScoreService{}, nil
		}),
		di.Provide[application.Policies](c, func(*di.Container) (application.Policies, error) {
			return application.Policies{"zero": domain.WeightedZero{}, "strict": domain.WeightedComplete{}}, nil
		}),
		di.Provide[*application.Facade](c, func(c *di.Container) (*application.Facade, error) {
			repo, err := di.Resolve[domain.CourseRepository](c)
			if err != nil {
				return nil, err
			}
			calculator, err := di.Resolve[domain.FinalScoreService](c)
			if err != nil {
				return nil, err
			}
			exporter, err := di.Resolve[application.ResultsExporter](c)
			if err != nil {
				return nil, err
			}
			policies, err := di.Resolve[application.Policies](c)
			if err != nil {
				return nil, err
			}
			return application.NewFacade(repo, calculator, exporter, policies), nil
		}),
		di.Provide[*console.UI](c, func(c *di.Container) (*console.UI, error) {
			app, err := di.Resolve[*application.Facade](c)
			if err != nil {
				return nil, err
			}
			return console.New(app, input, output, destination), nil
		}),
	}
	for _, err := range registrations {
		if err != nil {
			return nil, err
		}
	}
	return di.Resolve[*console.UI](c)
}
