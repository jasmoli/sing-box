package main

import (
	"time"

	"github.com/sagernet/sing-box/daemon"
	"github.com/sagernet/sing/common/byteformats"

	"github.com/spf13/cobra"
)

var commandAPIProviderShow = &cobra.Command{
	Use:   "show <provider>",
	Short: "Show an outbound provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAPIProviderShow(args[0])
	},
}

func init() {
	commandAPIProvider.AddCommand(commandAPIProviderShow)
}

func runAPIProviderShow(providerTag string) error {
	clientConn, client, err := createAPIClient()
	if err != nil {
		return err
	}
	defer clientConn.Close()
	provider, err := client.GetProvider(globalCtx, &daemon.ProviderRequest{Tag: providerTag})
	if err != nil {
		return err
	}
	block := blockWriter{}
	block.addLine("Tag", provider.GetTag())
	block.addLine("Vehicle", provider.GetVehicleType())
	block.addLine("Updated", formatProviderTime(provider.GetUpdatedAt()))
	if subscription := provider.GetSubscriptionInfo(); subscription != nil {
		block.addLine("Upload", byteformats.FormatBytes(uint64(subscription.GetUpload())))
		block.addLine("Download", byteformats.FormatBytes(uint64(subscription.GetDownload())))
		block.addLine("Total", byteformats.FormatBytes(uint64(subscription.GetTotal())))
		if subscription.GetExpire() > 0 {
			block.addLine("Expire", time.Unix(subscription.GetExpire(), 0).Local().Format(time.RFC3339))
		}
	}
	block.flush()
	table := tableWriter{
		header:       []string{"TAG", "TYPE", "DELAY"},
		emptyMessage: "no outbounds",
	}
	for _, outbound := range provider.GetOutbounds() {
		table.addRow(outbound.GetTag(), outbound.GetType(), formatDelay(outbound.GetUrlTestDelay()))
	}
	table.flush()
	return nil
}
