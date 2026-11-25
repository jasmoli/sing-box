package main

import (
	"time"

	F "github.com/sagernet/sing/common/format"

	"github.com/spf13/cobra"
)

var commandAPIProviderList = &cobra.Command{
	Use:   "list",
	Short: "List outbound providers",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAPIProviderList()
	},
}

func init() {
	commandAPIProvider.AddCommand(commandAPIProviderList)
}

func runAPIProviderList() error {
	clientConn, client, err := createAPIClient()
	if err != nil {
		return err
	}
	defer clientConn.Close()
	providers, err := fetchProviders(client)
	if err != nil {
		return err
	}
	table := tableWriter{
		header:       []string{"TAG", "VEHICLE", "UPDATED", "OUTBOUNDS"},
		emptyMessage: "no providers",
	}
	for _, provider := range providers {
		table.addRow(
			provider.GetTag(),
			provider.GetVehicleType(),
			formatProviderTime(provider.GetUpdatedAt()),
			F.ToString(len(provider.GetOutbounds())),
		)
	}
	table.flush()
	return nil
}

func formatProviderTime(timestamp int64) string {
	if timestamp <= 0 {
		return ""
	}
	return time.Unix(timestamp, 0).Local().Format(time.RFC3339)
}
