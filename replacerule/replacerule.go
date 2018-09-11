package replacerule

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type ReplaceRule struct {
	regex   *regexp.Regexp
	replace string
}

type RuleManager struct {
	rules []ReplaceRule
}

func GetManager(ruleString string) *RuleManager {
	manager := &RuleManager{}
	ruleStrs := strings.Split(ruleString, ",")
	for _, ruleStr := range ruleStrs {
		ruleStrArray := strings.Split(ruleStr, " ")
		if len(ruleStrArray) != 2 {
			continue
		}
		replaceRule := ReplaceRule{}
		regex, regErr := regexp.Compile(ruleStrArray[0])
		if regErr != nil {
			panic(fmt.Errorf("replace rule error: %v", regErr))
		}
		replaceRule.regex = regex
		replaceRule.replace = ruleStrArray[1]
		manager.rules = append(manager.rules, replaceRule)
	}

	sort.Slice(manager.rules, func(i int, j int) bool {
		iStr := manager.rules[i].regex.String()
		jStr := manager.rules[j].regex.String()
		if len(iStr) == len(jStr) {
			return jStr < iStr
		} else {
			return len(jStr) < len(iStr)
		}
	})
	return manager
}

func (m *RuleManager) Replace(path string) string {
	for _, rule := range m.rules {
		if rule.regex.MatchString(path) {
			return rule.regex.ReplaceAllString(path, rule.replace)
		}
	}
	return path
}
