package main

import (
	"github.com/sagernet/sing-box/daemon"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/emptypb"
)

var commandAPIRuleProvider = &cobra.Command{
	Use:   "rule-provider",
	Short: "Manage rule-set providers",
}

func init() {
	commandAPIRoot.AddCommand(commandAPIRuleProvider)
}

func fetchRuleProviders(client daemon.StartedServiceClient) ([]*daemon.RuleProvider, error) {
	providers, err := client.GetRuleProviders(globalCtx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	return providers.GetProviders(), nil
}
