package main

import (
	"github.com/sagernet/sing-box/daemon"

	"github.com/spf13/cobra"
)

var commandAPIRuleProviderUpdate = &cobra.Command{
	Use:   "update <provider>",
	Short: "Update a rule-set provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAPIRuleProviderUpdate(args[0])
	},
}

func init() {
	commandAPIRuleProvider.AddCommand(commandAPIRuleProviderUpdate)
}

func runAPIRuleProviderUpdate(providerTag string) error {
	clientConn, client, err := createAPIClient()
	if err != nil {
		return err
	}
	defer clientConn.Close()
	_, err = client.UpdateRuleProvider(globalCtx, &daemon.ProviderRequest{Tag: providerTag})
	return err
}
