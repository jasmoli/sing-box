package parser

import (
	"context"
	"strings"

	"github.com/sagernet/sing-box/option"
)

func ParseSubscription(ctx context.Context, content string) ([]option.Outbound, error) {
	if strings.Contains(content, "\"outbounds\"") {
		return ParseBoxSubscription(ctx, content)
	}
	if strings.Contains(content, "proxies") {
		return ParseClashSubscription(ctx, content)
	}
	return ParseRawSubscription(ctx, content)
}
