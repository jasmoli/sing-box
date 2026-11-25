package main

import (
	F "github.com/sagernet/sing/common/format"

	"github.com/spf13/cobra"
)

var commandAPIRuleList = &cobra.Command{
	Use:   "list",
	Short: "List routing rules",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAPIRuleList()
	},
}

func init() {
	commandAPIRule.AddCommand(commandAPIRuleList)
}

func runAPIRuleList() error {
	clientConn, client, err := createAPIClient()
	if err != nil {
		return err
	}
	defer clientConn.Close()
	rules, err := fetchRules(client)
	if err != nil {
		return err
	}
	table := tableWriter{
		header:       []string{"UUID", "TYPE", "PAYLOAD", "ACTION", "DISABLED"},
		emptyMessage: "no rules",
	}
	for _, rule := range rules {
		table.addRow(
			rule.GetUuid(),
			rule.GetType(),
			rule.GetPayload(),
			rule.GetProxy(),
			F.ToString(rule.GetDisabled()),
		)
	}
	table.flush()
	return nil
}
