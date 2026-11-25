package main

import (
	"github.com/sagernet/sing-box/daemon"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/emptypb"
)

var commandAPIProvider = &cobra.Command{
	Use:   "provider",
	Short: "Manage outbound providers",
}

func init() {
	commandAPIRoot.AddCommand(commandAPIProvider)
}

func fetchProviders(client daemon.StartedServiceClient) ([]*daemon.Provider, error) {
	providers, err := client.GetProviders(globalCtx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	return providers.GetProviders(), nil
}
