package filter

import (
	"regexp"
	"strconv"

	"github.com/sagernet/sing/common"
	E "github.com/sagernet/sing/common/exceptions"
	R "github.com/dlclark/regexp2"
)

func TestIncludes(tag string, includes []*regexp.Regexp) bool {
	if len(includes) == 0 {
		return true
	}
	return common.All(includes, func(it *regexp.Regexp) bool {
		matched := it.MatchString(tag)
		return matched
	})
}

func TestTypes(oType string, types []string) bool {
	if len(types) == 0 {
		return true
	}
	return common.Any(types, func(it string) bool {
		return oType == it
	})
}

func TestPorts(port uint16, ports map[uint16]bool) bool {
	if port == 0 || len(ports) == 0 {
		return true
	}
	_, ok := ports[port]
	return ok
}

func CreatePortsMap(ports []string) (map[uint16]bool, error) {
	portReg1 := R.MustCompile(`^\d+$`, R.None)
	portReg2 := R.MustCompile(`^(\d*):(\d*)$`, R.None)
	portMap := map[uint16]bool{}
	for i, portRaw := range ports {
		if matched, _ := portReg1.MatchString(portRaw); matched {
			port, _ := strconv.Atoi(portRaw)
			portMap[uint16(port)] = true
			continue
		}
		if portRaw == ":" {
			return nil, E.New("invalid ports item[", i, "]")
		}
		if match, _ := portReg2.FindStringMatch(portRaw); match != nil {
			start, _ := strconv.Atoi(match.Groups()[1].String())
			end, _ := strconv.Atoi(match.Groups()[2].String())
			if start < 0 || start > 65535 {
				return nil, E.New("invalid ports item[", i, "]")
			}
			if end < 0 || end > 65535 {
				return nil, E.New("invalid ports item[", i, "]")
			}
			if end == 0 {
				end = 65535
			}
			if start > end {
				return nil, E.New("invalid ports item[", i, "]")
			}
			for port := start; port <= end; port++ {
				portMap[uint16(port)] = true
			}
			continue
		}
		return nil, E.New("invalid ports item[", i, "]")
	}
	return portMap, nil
}
