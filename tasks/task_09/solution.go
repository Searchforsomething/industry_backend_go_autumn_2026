package main

import (
	"context"
	"errors"
	"sync"
)

type Result[R any] struct {
	Value R
	Err   error
}

var ErrInvalidWorkers = errors.New("workers must be positive")

func ParallelMap[T any, R any](ctx context.Context, workers int, in []T, fn func(context.Context, T) (R, error)) ([]Result[R], error) {
	if workers <= 0 {
		return nil, ErrInvalidWorkers
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(in) == 0 {
		return []Result[R]{}, nil
	}
	results := make([]Result[R], len(in))

	var wg sync.WaitGroup

	jobs := make(chan int)

	worker := func() {
		defer wg.Done()

		for i := range jobs {
			if ctx.Err() != nil {
				return
			}

			value, err := fn(ctx, in[i])

			if err != nil {
				var zero R
				value = zero
			}

			results[i] = Result[R]{
				Value: value,
				Err:   err,
			}
		}
	}

	workerCount := min(workers, len(in))

	for w := 1; w <= workerCount; w++ {
		wg.Add(1)
		go worker()
	}

dispatch:
	for i := range in {
		if ctx.Err() != nil {
			break dispatch
		}

		select {
		case <-ctx.Done():
			break dispatch
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
