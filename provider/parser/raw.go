package parser

import (
	"context"
	"strings"

	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
)

func ParseRawSubscription(_ context.Context, content string) ([]option.Outbound, error) {
	if base64Content, err := DecodeBase64URLSafe(content); err == nil && base64Content != content {
		servers, _ := parseRawSubscription(base64Content)
		if len(servers) > 0 {
			return servers, nil
		}
	}
	return parseRawSubscription(content)
}

func parseRawSubscription(content string) ([]option.Outbound, error) {
	var servers []option.Outbound
	content = strings.ReplaceAll(content, "\r\n", "\n")
	linkList := strings.Split(content, "\n")
	for _, linkLine := range linkList {
		linkLine = strings.TrimSpace(linkLine)
		if !strings.Contains(linkLine, "://") {
			continue
		}
		server, err := ParseSubscriptionLink(linkLine)
		if err != nil {
			continue
		}
		servers = append(servers, server)
	}
	if len(servers) == 0 {
		return nil, E.New("no servers found")
	}
	return servers, nil
}
