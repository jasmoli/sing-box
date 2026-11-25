package main

import (
	"github.com/sagernet/sing-box/daemon"

	"github.com/spf13/cobra"
)

var commandAPIRuleToggle = &cobra.Command{
	Use:   "toggle <uuid>",
	Short: "Enable or disable a routing rule",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAPIRuleToggle(args[0])
	},
}

func init() {
	commandAPIRule.AddCommand(commandAPIRuleToggle)
}

func runAPIRuleToggle(uuid string) error {
	clientConn, client, err := createAPIClient()
	if err != nil {
		return err
	}
	defer clientConn.Close()
	_, err = client.ChangeRuleStatus(globalCtx, &daemon.ChangeRuleStatusRequest{Uuid: uuid})
	return err
}
