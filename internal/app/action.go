package app

import (
	"context"
	"log/slog"

	"github.com/urfave/cli/v2"
)

// Runner is a generic function type for command runners.
type Runner[CONFIG any, RESULT any] func(context.Context, *slog.Logger, CONFIG, ...string) (RESULT, error)

var getLogger = GetLogger

// action is a generic action handler that executes a runner and outputs the result.
func action[C any, R any](ctx context.Context, c *cli.Context, cfg C, runner Runner[C, R]) error {
	logger := getLogger(c)

	result, err := runner(ctx, logger, cfg, c.Args().Slice()...)
	if err != nil {
		return err
	}

	return output(c.App.Writer, result)
}

// Default creates an action function with pre-bound config and runner.
func Default[C any, R any](cfg *C, runner Runner[C, R]) func(*cli.Context) error {
	return func(c *cli.Context) error {
		return action(c.Context, c, *cfg, runner)
	}
}
