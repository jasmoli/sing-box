package main

import (
	"github.com/sagernet/sing-box/daemon"

	"github.com/spf13/cobra"
)

var commandAPIProviderUpdate = &cobra.Command{
	Use:   "update <provider>",
	Short: "Update an outbound provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAPIProviderUpdate(args[0])
	},
}

func init() {
	commandAPIProvider.AddCommand(commandAPIProviderUpdate)
}

func runAPIProviderUpdate(providerTag string) error {
	clientConn, client, err := createAPIClient()
	if err != nil {
		return err
	}
	defer clientConn.Close()
	_, err = client.UpdateProvider(globalCtx, &daemon.ProviderRequest{Tag: providerTag})
	return err
}
