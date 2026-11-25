package main

import (
	F "github.com/sagernet/sing/common/format"

	"github.com/spf13/cobra"
)

var commandAPIRuleProviderList = &cobra.Command{
	Use:   "list",
	Short: "List rule-set providers",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAPIRuleProviderList()
	},
}

func init() {
	commandAPIRuleProvider.AddCommand(commandAPIRuleProviderList)
}

func runAPIRuleProviderList() error {
	clientConn, client, err := createAPIClient()
	if err != nil {
		return err
	}
	defer clientConn.Close()
	providers, err := fetchRuleProviders(client)
	if err != nil {
		return err
	}
	table := tableWriter{
		header:       []string{"TAG", "VEHICLE", "BEHAVIOR", "RULES", "UPDATED"},
		emptyMessage: "no rule providers",
	}
	for _, provider := range providers {
		table.addRow(
			provider.GetTag(),
			provider.GetVehicleType(),
			provider.GetBehavior(),
			F.ToString(provider.GetRuleCount()),
			formatProviderTime(provider.GetUpdatedAt()),
		)
	}
	table.flush()
	return nil
}
