package services

import (
	"fmt"
	"regexp"
	"strings"
)

func ParseFilter(filter string) (string, []interface{}) {
	var params []interface{}

	operandMap := map[string]string{
		" eq ":  " = ",
		" ne ":  " != ",
		" gt ":  " > ",
		" ge ":  " >= ",
		" lt ":  " < ",
		" le ":  " <= ",
		" and ": " AND ",
		" or ":  " OR ",
	}
	for key, value := range operandMap {
		filter = strings.ReplaceAll(filter, key, value)
	}

	propertyComparisonRegex := regexp.MustCompile(`properties\.([\w\.]+)\s*(=|!=)\s*['"]([^'"]*)['"]`)
	filter = propertyComparisonRegex.ReplaceAllStringFunc(filter, func(match string) string {
		matches := propertyComparisonRegex.FindStringSubmatch(match)
		if len(matches) != 4 {
			return match
		}
		key := matches[1]
		operator := matches[2]
		value := matches[3]

		jsonFragment := fmt.Sprintf(`{"%s": ["%s"]}`, key, value)
		sqlExpr := `properties @> ?::jsonb`

		if operator == "!=" {
			sqlExpr = fmt.Sprintf(`NOT (%s)`, sqlExpr)
		}

		params = append(params, jsonFragment)

		return sqlExpr
	})

	funcRegex := regexp.MustCompile(`(?i)(startswith|contains|endswith)\s*\(\s*(\w+)\s*,\s*['"]([^'"]*)['"]\s*\)`)
	filter = funcRegex.ReplaceAllStringFunc(filter, func(match string) string {
		matches := funcRegex.FindStringSubmatch(match)
		if len(matches) != 4 {
			return match
		}
		function := strings.ToLower(matches[1]) // Function name
		field := matches[2]                     // Field name
		value := matches[3]                     // Value inside quotes

		var sqlExpr string
		switch function {
		case "startswith":
			params = append(params, value+"%")
			sqlExpr = fmt.Sprintf("%s LIKE ?", field)
		case "contains":
			params = append(params, "%"+value+"%")
			sqlExpr = fmt.Sprintf("%s LIKE ?", field)
		case "endswith":
			params = append(params, "%"+value)
			sqlExpr = fmt.Sprintf("%s LIKE ?", field)
		default:
			return match
		}

		return sqlExpr
	})

	re := regexp.MustCompile(`['"]([^'"]*)['"]`)
	filter = re.ReplaceAllStringFunc(filter, func(match string) string {
		value := strings.Trim(match, `'"`)
		params = append(params, value)
		return "?"
	})

	return filter, params
}
