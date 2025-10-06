package filter

import (
	"regexp"

	"github.com/sagernet/sing/common"
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