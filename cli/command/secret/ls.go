package secret

import (
	"context"
	"sort"

	"github.com/yuyangjack/dockercli/cli"
	"github.com/yuyangjack/dockercli/cli/command"
	"github.com/yuyangjack/dockercli/cli/command/formatter"
	"github.com/yuyangjack/dockercli/opts"
	"github.com/yuyangjack/moby/api/types"
	"github.com/spf13/cobra"
	"vbom.ml/util/sortorder"
)

type listOptions struct {
	quiet  bool
	format string
	filter opts.FilterOpt
}

func newSecretListCommand(dockerCli command.Cli) *cobra.Command {
	options := listOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Aliases: []string{"list"},
		Short:   "List secrets",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSecretList(dockerCli, options)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only display IDs")
	flags.StringVarP(&options.format, "format", "", "", "Pretty-print secrets using a Go template")
	flags.VarP(&options.filter, "filter", "f", "Filter output based on conditions provided")

	return cmd
}

func runSecretList(dockerCli command.Cli, options listOptions) error {
	client := dockerCli.Client()
	ctx := context.Background()

	secrets, err := client.SecretList(ctx, types.SecretListOptions{Filters: options.filter.Value()})
	if err != nil {
		return err
	}
	format := options.format
	if len(format) == 0 {
		if len(dockerCli.ConfigFile().SecretFormat) > 0 && !options.quiet {
			format = dockerCli.ConfigFile().SecretFormat
		} else {
			format = formatter.TableFormatKey
		}
	}

	sort.Slice(secrets, func(i, j int) bool {
		return sortorder.NaturalLess(secrets[i].Spec.Name, secrets[j].Spec.Name)
	})

	secretCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: formatter.NewSecretFormat(format, options.quiet),
	}
	return formatter.SecretWrite(secretCtx, secrets)
}
