package stack

import (
	"github.com/yuyangjack/dockercli/cli"
	"github.com/yuyangjack/dockercli/cli/command"
	"github.com/yuyangjack/dockercli/cli/command/stack/kubernetes"
	"github.com/yuyangjack/dockercli/cli/command/stack/options"
	"github.com/yuyangjack/dockercli/cli/command/stack/swarm"
	cliopts "github.com/yuyangjack/dockercli/opts"
	"github.com/spf13/cobra"
)

func newPsCommand(dockerCli command.Cli, common *commonOptions) *cobra.Command {
	opts := options.PS{Filter: cliopts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "ps [OPTIONS] STACK",
		Short: "List the tasks in the stack",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Namespace = args[0]
			if err := validateStackName(opts.Namespace); err != nil {
				return err
			}

			switch {
			case common.orchestrator.HasAll():
				return errUnsupportedAllOrchestrator
			case common.orchestrator.HasKubernetes():
				kli, err := kubernetes.WrapCli(dockerCli, kubernetes.NewOptions(cmd.Flags(), common.orchestrator))
				if err != nil {
					return err
				}
				return kubernetes.RunPS(kli, opts)
			default:
				return swarm.RunPS(dockerCli, opts)
			}
		},
	}
	flags := cmd.Flags()
	flags.BoolVar(&opts.NoTrunc, "no-trunc", false, "Do not truncate output")
	flags.BoolVar(&opts.NoResolve, "no-resolve", false, "Do not map IDs to Names")
	flags.VarP(&opts.Filter, "filter", "f", "Filter output based on conditions provided")
	flags.BoolVarP(&opts.Quiet, "quiet", "q", false, "Only display task IDs")
	flags.StringVar(&opts.Format, "format", "", "Pretty-print tasks using a Go template")
	kubernetes.AddNamespaceFlag(flags)
	return cmd
}
