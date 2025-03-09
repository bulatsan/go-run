package runner

import (
	"context"
	"sync"
)

type Runner interface {
	Run(context.Context) error
}

type runner struct {
	fn func(context.Context) error
}

func (r *runner) Run(ctx context.Context) error {
	return r.fn(ctx)
}

func New(fn func(context.Context) error) Runner {
	return &runner{fn: fn}
}

func Err(err error) Runner {
	return New(func(context.Context) error { return err })
}

func OK() Runner {
	return Err(nil)
}

func Join(runners ...Runner) Runner {
	if len(runners) == 0 {
		return OK()
	}

	return New(func(ctx context.Context) error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		var (
			wg   sync.WaitGroup
			err  error
			once sync.Once
		)

		for _, runner := range runners {
			wg.Add(1)

			go func(runner Runner) {
				defer wg.Done()

				if rerr := runner.Run(ctx); rerr != nil {
					once.Do(func() {
						cancel()
						err = rerr
					})
				}
			}(runner)
		}

		wg.Wait()
		return err
	})
}
