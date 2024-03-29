package container

import (
	"context"

	"github.com/yuyangjack/dockercli/cli"
	"github.com/yuyangjack/dockercli/cli/command"
	"github.com/yuyangjack/dockercli/cli/command/inspect"
	"github.com/spf13/cobra"
)

type inspectOptions struct {
	format string
	size   bool
	refs   []string
}

// newInspectCommand creates a new cobra.Command for `docker container inspect`
func newInspectCommand(dockerCli command.Cli) *cobra.Command {
	var opts inspectOptions

	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] CONTAINER [CONTAINER...]",
		Short: "Display detailed information on one or more containers",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.refs = args
			return runInspect(dockerCli, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.format, "format", "f", "", "Format the output using the given Go template")
	flags.BoolVarP(&opts.size, "size", "s", false, "Display total file sizes")

	return cmd
}

func runInspect(dockerCli command.Cli, opts inspectOptions) error {
	client := dockerCli.Client()
	ctx := context.Background()

	getRefFunc := func(ref string) (interface{}, []byte, error) {
		return client.ContainerInspectWithRaw(ctx, ref, opts.size)
	}
	return inspect.Inspect(dockerCli.Out(), opts.refs, opts.format, getRefFunc)
}
