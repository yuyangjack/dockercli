package engine

import (
	"context"

	"github.com/yuyangjack/dockercli/cli/command"
	"github.com/yuyangjack/dockercli/cli/trust"
	clitypes "github.com/yuyangjack/dockercli/types"
	"github.com/yuyangjack/distribution/reference"
	"github.com/yuyangjack/moby/api/types"
	registrytypes "github.com/yuyangjack/moby/api/types/registry"
	"github.com/pkg/errors"
)

func getRegistryAuth(cli command.Cli, registryPrefix string) (*types.AuthConfig, error) {
	if registryPrefix == "" {
		registryPrefix = clitypes.RegistryPrefix
	}
	distributionRef, err := reference.ParseNormalizedNamed(registryPrefix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse image name: %s", registryPrefix)
	}
	imgRefAndAuth, err := trust.GetImageReferencesAndAuth(context.Background(), nil, authResolver(cli), distributionRef.String())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get imgRefAndAuth")
	}
	return imgRefAndAuth.AuthConfig(), nil
}

func authResolver(cli command.Cli) func(ctx context.Context, index *registrytypes.IndexInfo) types.AuthConfig {
	return func(ctx context.Context, index *registrytypes.IndexInfo) types.AuthConfig {
		return command.ResolveAuthConfig(ctx, cli, index)
	}
}
