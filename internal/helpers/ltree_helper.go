package helpers

import (
	"regexp"
	"strings"
)

func SanitizeLtreeIdentifier(input string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	return re.ReplaceAllString(input, "_")
}

var nonAlphaNumericRegex = regexp.MustCompile(`[^a-zA-Z0-9_]`)

func PathToLtree(path string) string {
	p := strings.ReplaceAll(path, "/", ".")

	p = nonAlphaNumericRegex.ReplaceAllString(p, "_")
	return p
}

func LtreeToPath(ltree string) string {
	return strings.ReplaceAll(ltree, ".", "/")
}
