package image

import (
	"context"
	"io"
	"os"

	"github.com/yuyangjack/dockercli/cli"
	"github.com/yuyangjack/dockercli/cli/command"
	dockeropts "github.com/yuyangjack/dockercli/opts"
	"github.com/yuyangjack/moby/api/types"
	"github.com/yuyangjack/moby/pkg/jsonmessage"
	"github.com/yuyangjack/moby/pkg/urlutil"
	"github.com/spf13/cobra"
)

type importOptions struct {
	source    string
	reference string
	changes   dockeropts.ListOpts
	message   string
	platform  string
}

// NewImportCommand creates a new `docker import` command
func NewImportCommand(dockerCli command.Cli) *cobra.Command {
	var options importOptions

	cmd := &cobra.Command{
		Use:   "import [OPTIONS] file|URL|- [REPOSITORY[:TAG]]",
		Short: "Import the contents from a tarball to create a filesystem image",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.source = args[0]
			if len(args) > 1 {
				options.reference = args[1]
			}
			return runImport(dockerCli, options)
		},
	}

	flags := cmd.Flags()

	options.changes = dockeropts.NewListOpts(nil)
	flags.VarP(&options.changes, "change", "c", "Apply Dockerfile instruction to the created image")
	flags.StringVarP(&options.message, "message", "m", "", "Set commit message for imported image")
	command.AddPlatformFlag(flags, &options.platform)

	return cmd
}

func runImport(dockerCli command.Cli, options importOptions) error {
	var (
		in      io.Reader
		srcName = options.source
	)

	if options.source == "-" {
		in = dockerCli.In()
	} else if !urlutil.IsURL(options.source) {
		srcName = "-"
		file, err := os.Open(options.source)
		if err != nil {
			return err
		}
		defer file.Close()
		in = file
	}

	source := types.ImageImportSource{
		Source:     in,
		SourceName: srcName,
	}

	importOptions := types.ImageImportOptions{
		Message:  options.message,
		Changes:  options.changes.GetAll(),
		Platform: options.platform,
	}

	clnt := dockerCli.Client()

	responseBody, err := clnt.ImageImport(context.Background(), source, options.reference, importOptions)
	if err != nil {
		return err
	}
	defer responseBody.Close()

	return jsonmessage.DisplayJSONMessagesToStream(responseBody, dockerCli.Out(), nil)
}
