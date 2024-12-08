package cmd

import "regexp"

func SanitizeLtreeIdentifier(input string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	return re.ReplaceAllString(input, "_")
}
