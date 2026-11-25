package main

import (
	"github.com/sagernet/sing-box/daemon"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/emptypb"
)

var commandAPIRule = &cobra.Command{
	Use:   "rule",
	Short: "Manage routing rules",
}

func init() {
	commandAPIRoot.AddCommand(commandAPIRule)
}

func fetchRules(client daemon.StartedServiceClient) ([]*daemon.Rule, error) {
	rules, err := client.GetRules(globalCtx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	return rules.GetRules(), nil
}
