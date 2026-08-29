package parser

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/sagernet/sing/common/byteformats"
	"github.com/sagernet/sing/common/json/badoption"
)

func stringToUint16(content string) uint16 {
	intNum, _ := strconv.Atoi(content)
	return uint16(intNum)
}

func stringToInt64(content string) int64 {
	intNum, _ := strconv.Atoi(content)
	return int64(intNum)
}

func stringToUint32(content string) uint32 {
	intNum, _ := strconv.Atoi(content)
	return uint32(intNum)
}

func decodeURIComponent(content string) string {
	result, _ := url.QueryUnescape(content)
	return result
}

func trimBlank(str string) string {
	str = strings.Trim(str, " ")
	str = strings.Trim(str, "\a")
	str = strings.Trim(str, "\b")
	str = strings.Trim(str, "\f")
	str = strings.Trim(str, "\r")
	str = strings.Trim(str, "\t")
	str = strings.Trim(str, "\v")
	return str
}

func splitKeyValueWithEqual(content string) (string, string) {
	if !strings.Contains(content, "=") {
		return trimBlank(content), "1"
	}
	arr := strings.SplitN(content, "=", 2)
	return trimBlank(arr[0]), trimBlank(arr[1])
}

func splitKeyValueWithColon(content string) (string, string) {
	if !strings.Contains(content, ":") {
		return trimBlank(content), "1"
	}
	arr := strings.Split(content, ":")
	return trimBlank(arr[0]), trimBlank(arr[1])
}

func DecodeBase64Safe(content string) string {
	if decode, err := base64.StdEncoding.DecodeString(content); err == nil {
		return string(decode)
	}
	if decode, err := base64.RawStdEncoding.DecodeString(content); err == nil {
		return string(decode)
	}
	if decode, err := base64.URLEncoding.DecodeString(content); err == nil {
		return string(decode)
	}
	if decode, err := base64.RawURLEncoding.DecodeString(content); err == nil {
		return string(decode)
	}
	return content
}

func DecodeBase64URLSafe(content string) (string, error) {
	s := strings.ReplaceAll(content, " ", "-")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "+", "-")
	s = strings.ReplaceAll(s, "=", "")
	result, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return content, nil
	}
	return string(result), nil
}

func getFirstLine(content string) (string, string) {
	lines := strings.Split(content, "\n")
	if len(lines) == 1 {
		return lines[0], ""
	}
	others := strings.Join(lines[1:], "\n")
	return lines[0], others
}

func GetFirstLine(content string) (string, string) {
	return getFirstLine(content)
}

func stringToNetworkBytes(content string) *byteformats.NetworkBytesCompat {
	if content == "" {
		return nil
	}
	var value byteformats.NetworkBytesCompat
	data, _ := json.Marshal(content)
	if err := value.UnmarshalJSON(data); err == nil {
		return &value
	}
	return nil
}

func clashPorts(ports string) badoption.Listable[string] {
	if ports == "" {
		return nil
	}
	serverPorts := badoption.Listable[string]{}
	ports = strings.ReplaceAll(ports, "/", ",")
	for port := range strings.SplitSeq(ports, ",") {
		if port == "" {
			continue
		}
		port = strings.Replace(port, "-", ":", 1)
		serverPorts = append(serverPorts, port)
	}
	return serverPorts
}
