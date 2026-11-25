package main

import (
	"github.com/sagernet/sing-box/daemon"
	F "github.com/sagernet/sing/common/format"

	"github.com/spf13/cobra"
)

var commandAPIRuleProviderShow = &cobra.Command{
	Use:   "show <provider>",
	Short: "Show a rule-set provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAPIRuleProviderShow(args[0])
	},
}

func init() {
	commandAPIRuleProvider.AddCommand(commandAPIRuleProviderShow)
}

func runAPIRuleProviderShow(providerTag string) error {
	clientConn, client, err := createAPIClient()
	if err != nil {
		return err
	}
	defer clientConn.Close()
	provider, err := client.GetRuleProvider(globalCtx, &daemon.ProviderRequest{Tag: providerTag})
	if err != nil {
		return err
	}
	block := blockWriter{}
	block.addLine("Tag", provider.GetTag())
	block.addLine("Vehicle", provider.GetVehicleType())
	block.addLine("Behavior", provider.GetBehavior())
	block.addLine("Rules", F.ToString(provider.GetRuleCount()))
	block.addLine("Updated", formatProviderTime(provider.GetUpdatedAt()))
	block.flush()
	return nil
}
