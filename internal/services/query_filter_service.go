package services

import (
	"fmt"
	"regexp"
	"strings"
)

func ParseFilter(filter string) (string, []interface{}) {
	var params []interface{}

	// Step 1: Replace logical operators
	logicalOperators := map[string]string{
		" and ": " AND ",
		" or ":  " OR ",
	}
	for key, value := range logicalOperators {
		filter = strings.ReplaceAll(filter, key, value)
	}

	// Step 2: Process comparison expressions
	comparisonRegex := regexp.MustCompile(`(?i)(properties\.\w+|\w+)\s+(eq|ne|gt|ge|lt|le|startswith|contains|endswith)\s+['"]([^'"]*)['"]`)
	filter = comparisonRegex.ReplaceAllStringFunc(filter, func(match string) string {
		matches := comparisonRegex.FindStringSubmatch(match)
		if len(matches) != 4 {
			return match
		}
		column := matches[1]
		operator := strings.ToLower(matches[2])
		value := matches[3]

		var sqlExpr string

		if strings.HasPrefix(column, "properties.") {
			// Handle properties
			key := strings.TrimPrefix(column, "properties.")

			switch operator {
			case "eq":
				sqlExpr = `properties @> ?::jsonb`
				jsonFragment := fmt.Sprintf(`{"%s": ["%s"]}`, key, value)
				params = append(params, jsonFragment)
			case "ne":
				sqlExpr = `NOT (properties @> ?::jsonb)`
				jsonFragment := fmt.Sprintf(`{"%s": ["%s"]}`, key, value)
				params = append(params, jsonFragment)
			case "contains":
				sqlExpr = fmt.Sprintf(`EXISTS (SELECT 1 FROM jsonb_array_elements_text(properties->'%s') AS elems WHERE elems LIKE ?)`, key)
				params = append(params, "%"+value+"%")
			case "startswith":
				sqlExpr = fmt.Sprintf(`EXISTS (SELECT 1 FROM jsonb_array_elements_text(properties->'%s') AS elems WHERE elems LIKE ?)`, key)
				params = append(params, value+"%")
			case "endswith":
				sqlExpr = fmt.Sprintf(`EXISTS (SELECT 1 FROM jsonb_array_elements_text(properties->'%s') AS elems WHERE elems LIKE ?)`, key)
				params = append(params, "%"+value)
			default:
				// If operator is not recognized, return the match unchanged
				return match
			}
		} else {
			// Handle normal columns
			switch operator {
			case "eq":
				sqlExpr = fmt.Sprintf("%s = ?", column)
				params = append(params, value)
			case "ne":
				sqlExpr = fmt.Sprintf("%s != ?", column)
				params = append(params, value)
			case "gt":
				sqlExpr = fmt.Sprintf("%s > ?", column)
				params = append(params, value)
			case "ge":
				sqlExpr = fmt.Sprintf("%s >= ?", column)
				params = append(params, value)
			case "lt":
				sqlExpr = fmt.Sprintf("%s < ?", column)
				params = append(params, value)
			case "le":
				sqlExpr = fmt.Sprintf("%s <= ?", column)
				params = append(params, value)
			case "startswith":
				sqlExpr = fmt.Sprintf("%s LIKE ?", column)
				params = append(params, value+"%")
			case "contains":
				sqlExpr = fmt.Sprintf("%s LIKE ?", column)
				params = append(params, "%"+value+"%")
			case "endswith":
				sqlExpr = fmt.Sprintf("%s LIKE ?", column)
				params = append(params, "%"+value)
			default:
				// If operator is not recognized, return the match unchanged
				return match
			}
		}
		return sqlExpr
	})

	// Step 3: Replace any remaining literals with placeholders
	re := regexp.MustCompile(`['"]([^'"]*)['"]`)
	filter = re.ReplaceAllStringFunc(filter, func(match string) string {
		value := strings.Trim(match, `'"`)
		params = append(params, value)
		return "?"
	})

	return filter, params
}
