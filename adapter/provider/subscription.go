package provider

import (
	"regexp"
	"strconv"

	"github.com/sagernet/sing-box/adapter"
)

func ParseInfo(infoStr string) (adapter.SubscriptionInfo, bool) {
	info := adapter.SubscriptionInfo{}
	if infoStr == "" {
		return info, false
	}
	reg := regexp.MustCompile(`(upload|download|total|expire)[\s\t]*=[\s\t]*(-?\d*);?`)
	matches := reg.FindAllStringSubmatch(infoStr, 4)
	if len(matches) == 0 {
		return info, false
	}
	for _, match := range matches {
		key, value := match[1], match[2]
		var parsed int64
		if value == "" {
			continue
		}
		parsed, _ = strconv.ParseInt(value, 10, 64)
		switch key {
		case "upload":
			info.Upload = parsed
		case "download":
			info.Download = parsed
		case "total":
			info.Total = parsed
		case "expire":
			info.Expire = parsed
		default:
			return info, false
		}
	}
	return info, true
}
