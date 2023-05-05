package runner

import (
	"github.com/23technologies/23kectl/pkg/check"
	"math/rand"
	"time"
)

type Runner struct {
	checks []check.Check
}

func (runner *Runner) AddCheck(checks ...check.Check) {
	runner.checks = append(runner.checks, checks...)
}

func (runner *Runner) RunAllOnce() {
	for _, c := range runner.checks {
		runner.RunOnce(c)
	}
}

func (runner *Runner) RunAllOnceAsync() chan []*check.Result {
	ch := make(chan []*check.Result)

	results := make([]*check.Result, len(runner.checks))
	done := make([]bool, len(runner.checks))

	for i, c := range runner.checks {
		go func(i int) {
			time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
			results[i] = runner.RunOnce(c)
			ch <- results
			done[i] = true

			for _, d := range done {
				if !d {
					return
				}
			}

			close(ch)
		}(i)
	}

	return ch
}

func (runner *Runner) RunOnce(c check.Check) *check.Result {
	result := c.Run()

	if result.IsError {
		if withErrCallback, ok := c.(check.WithOnError); ok {
			withErrCallback.OnError()
		}
	}

	return result
}

func New() *Runner {
	return &Runner{}
}
