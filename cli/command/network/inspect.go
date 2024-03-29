package network

import (
	"context"

	"github.com/yuyangjack/dockercli/cli"
	"github.com/yuyangjack/dockercli/cli/command"
	"github.com/yuyangjack/dockercli/cli/command/inspect"
	"github.com/yuyangjack/moby/api/types"
	"github.com/spf13/cobra"
)

type inspectOptions struct {
	format  string
	names   []string
	verbose bool
}

func newInspectCommand(dockerCli command.Cli) *cobra.Command {
	var opts inspectOptions

	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] NETWORK [NETWORK...]",
		Short: "Display detailed information on one or more networks",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.names = args
			return runInspect(dockerCli, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.format, "format", "f", "", "Format the output using the given Go template")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "Verbose output for diagnostics")

	return cmd
}

func runInspect(dockerCli command.Cli, opts inspectOptions) error {
	client := dockerCli.Client()

	ctx := context.Background()

	getNetFunc := func(name string) (interface{}, []byte, error) {
		return client.NetworkInspectWithRaw(ctx, name, types.NetworkInspectOptions{Verbose: opts.verbose})
	}

	return inspect.Inspect(dockerCli.Out(), opts.names, opts.format, getNetFunc)
}
