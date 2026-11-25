package main

import (
	"slices"
	"time"

	"github.com/sagernet/sing-box/daemon"

	"github.com/spf13/cobra"
)

var (
	commandAPIProviderHealthCheckFlagURL     string
	commandAPIProviderHealthCheckFlagTimeout time.Duration
)

var commandAPIProviderHealthCheck = &cobra.Command{
	Use:   "healthcheck <provider> [outbound]",
	Short: "Run a health check for an outbound provider",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAPIProviderHealthCheck(args)
	},
}

func init() {
	commandAPIProviderHealthCheck.Flags().StringVar(&commandAPIProviderHealthCheckFlagURL, "url", "", "Test URL")
	commandAPIProviderHealthCheck.Flags().DurationVar(&commandAPIProviderHealthCheckFlagTimeout, "timeout", 5*time.Second, "Test timeout")
	commandAPIProvider.AddCommand(commandAPIProviderHealthCheck)
}

func runAPIProviderHealthCheck(args []string) error {
	clientConn, client, err := createAPIClient()
	if err != nil {
		return err
	}
	defer clientConn.Close()
	if len(args) == 2 {
		result, err := client.HealthCheckProviderOutbound(globalCtx, &daemon.ProviderOutboundRequest{
			Tag:         args[0],
			OutboundTag: args[1],
			Url:         commandAPIProviderHealthCheckFlagURL,
			Timeout:     int32(commandAPIProviderHealthCheckFlagTimeout.Milliseconds()),
		})
		if err != nil {
			return err
		}
		table := tableWriter{
			header: []string{"TAG", "DELAY"},
		}
		table.addRow(args[1], formatDelay(result.GetDelays()[args[1]]))
		table.flush()
		return nil
	}
	result, err := client.HealthCheckProvider(globalCtx, &daemon.ProviderRequest{Tag: args[0]})
	if err != nil {
		return err
	}
	table := tableWriter{
		header:       []string{"TAG", "DELAY"},
		emptyMessage: "no outbounds",
	}
	delayTags := make([]string, 0, len(result.GetDelays()))
	for tag := range result.GetDelays() {
		delayTags = append(delayTags, tag)
	}
	slices.Sort(delayTags)
	for _, tag := range delayTags {
		table.addRow(tag, formatDelay(result.GetDelays()[tag]))
	}
	table.flush()
	return nil
}
