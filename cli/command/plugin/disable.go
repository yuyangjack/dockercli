package plugin

import (
	"context"
	"fmt"

	"github.com/yuyangjack/dockercli/cli"
	"github.com/yuyangjack/dockercli/cli/command"
	"github.com/yuyangjack/moby/api/types"
	"github.com/spf13/cobra"
)

func newDisableCommand(dockerCli command.Cli) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "disable [OPTIONS] PLUGIN",
		Short: "Disable a plugin",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDisable(dockerCli, args[0], force)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&force, "force", "f", false, "Force the disable of an active plugin")
	return cmd
}

func runDisable(dockerCli command.Cli, name string, force bool) error {
	if err := dockerCli.Client().PluginDisable(context.Background(), name, types.PluginDisableOptions{Force: force}); err != nil {
		return err
	}
	fmt.Fprintln(dockerCli.Out(), name)
	return nil
}
