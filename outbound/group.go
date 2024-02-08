package outbound

import (
	"context"
	"regexp"
	"strconv"

	"github.com/sagernet/sing-box/adapter"
	E "github.com/sagernet/sing/common/exceptions"
)

type myGroupAdapter struct {
	ctx             context.Context
	tags            []string
	uses            []string
	useAllProviders bool
	includes        []string
	excludes        string
	types           []string
	ports           map[int]bool
	providers       map[string]adapter.OutboundProvider
}

func CreatePortsMap(ports []string) (map[int]bool, error) {
	portReg1 := regexp.MustCompile(`^\d+$`)
	portReg2 := regexp.MustCompile(`^(\d*):(\d*)$`)
	portMap := map[int]bool{}
	for i, portRaw := range ports {
		result := portReg1.FindStringSubmatch(portRaw)
		if len(result) > 0 {
			port, _ := strconv.Atoi(portRaw)
			if port < 0 || port > 65535 {
				return nil, E.New("invalid ports item[", i, "]")
			}
			portMap[port] = true
			continue
		}
		if portRaw == ":" {
			return nil, E.New("invalid ports item[", i, "]")
		}
		result = portReg2.FindStringSubmatch(portRaw)
		if len(result) > 0 {
			start, _ := strconv.Atoi(result[1])
			end, _ := strconv.Atoi(result[2])
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
				portMap[port] = true
			}
			continue
		}
		return nil, E.New("invalid ports item[", i, "]")
	}
	return portMap, nil
}

func (s *myGroupAdapter) OutboundFilter(outbound adapter.Outbound) bool {
	tag := outbound.Tag()
	oType := outbound.Type()
	port := outbound.Port()
	return s.TestIncludes(tag) && s.TestExcludes(tag) && s.TestTypes(oType) && s.TestPorts(port)
}

func (s *myGroupAdapter) TestIncludes(tag string) bool {
	for _, filter := range s.includes {
		reg := regexp.MustCompile("(?i)" + filter)
		if len(reg.FindStringIndex(tag)) == 0 {
			return false
		}
	}
	return true
}

func (s *myGroupAdapter) TestExcludes(tag string) bool {
	filter := s.excludes
	if filter == "" {
		return true
	}
	reg := regexp.MustCompile("(?i)" + filter)
	return len(reg.FindStringIndex(tag)) == 0
}

func (s *myGroupAdapter) TestTypes(oType string) bool {
	if len(s.types) == 0 {
		return true
	}
	for _, iType := range s.types {
		if oType == iType {
			return true
		}
	}
	return false
}

func (s *myGroupAdapter) TestPorts(port int) bool {
	if len(s.ports) == 0 {
		return true
	}
	_, ok := s.ports[port]
	return ok
}
