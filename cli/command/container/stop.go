package container

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yuyangjack/dockercli/cli"
	"github.com/yuyangjack/dockercli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type stopOptions struct {
	time        int
	timeChanged bool

	containers []string
}

// NewStopCommand creates a new cobra.Command for `docker stop`
func NewStopCommand(dockerCli command.Cli) *cobra.Command {
	var opts stopOptions

	cmd := &cobra.Command{
		Use:   "stop [OPTIONS] CONTAINER [CONTAINER...]",
		Short: "Stop one or more running containers",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containers = args
			opts.timeChanged = cmd.Flags().Changed("time")
			return runStop(dockerCli, &opts)
		},
	}

	flags := cmd.Flags()
	flags.IntVarP(&opts.time, "time", "t", 10, "Seconds to wait for stop before killing it")
	return cmd
}

func runStop(dockerCli command.Cli, opts *stopOptions) error {
	ctx := context.Background()

	var timeout *time.Duration
	if opts.timeChanged {
		timeoutValue := time.Duration(opts.time) * time.Second
		timeout = &timeoutValue
	}

	var errs []string

	errChan := parallelOperation(ctx, opts.containers, func(ctx context.Context, id string) error {
		return dockerCli.Client().ContainerStop(ctx, id, timeout)
	})
	for _, container := range opts.containers {
		if err := <-errChan; err != nil {
			errs = append(errs, err.Error())
			continue
		}
		fmt.Fprintln(dockerCli.Out(), container)
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
