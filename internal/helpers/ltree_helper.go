package helpers

import (
	"Boxed/internal/models"
	"strings"
)

// PathToLtree converts user path to ltree format
// Example: "native-1234/file2.pkg" -> "native_1234.file2_pkg"
func PathToLtree(path string) string {
	// First split by slashes to get path parts
	parts := strings.Split(path, "/")

	// Process each part
	for i, part := range parts {
		// Only replace hyphens with underscores
		parts[i] = strings.ReplaceAll(part, "-", "_")
	}

	// Join with dots
	return strings.Join(parts, ".")
}

func LtreeToUserPath(item *models.Item) string {
	if item == nil {
		return ""
	}

	// Split the path into parts
	parts := strings.Split(item.Path, ".")

	if item.Type == "file" {
		// Get the last parts that might make up the filename
		lastParts := parts[len(parts)-2:] // Get last two parts
		possibleFilename := strings.Join(lastParts, ".")

		// If these parts match our filename, we should exclude them from path
		if possibleFilename == item.Name {
			// Use all parts except the last two
			parentParts := parts[:len(parts)-2]

			// Convert parent parts from underscore to hyphen
			for i := range parentParts {
				parentParts[i] = strings.ReplaceAll(parentParts[i], "_", "-")
			}

			// Join with the original filename
			if len(parentParts) > 0 {
				return strings.Join(parentParts, "/") + "/" + item.Name
			}
			return item.Name
		}
	}

	// For folders or non-matching files, convert all parts
	for i := range parts {
		parts[i] = strings.ReplaceAll(parts[i], "_", "-")
	}

	return strings.Join(parts, "/")
}

// UserPathToLtree converts a user path to database format
func UserPathToLtree(path string) string {
	parts := strings.Split(path, "/")

	// Convert all parts to underscore format
	for i := range parts {
		if i < len(parts)-1 { // Don't convert the last part if it's a filename
			parts[i] = strings.ReplaceAll(parts[i], "-", "_")
		}
	}

	return strings.Join(parts, ".")
}

// SanitizeLtreeIdentifier converts a string to ltree-safe format
func SanitizeLtreeIdentifier(input string) string {
	// Replace hyphens with underscores
	return strings.ReplaceAll(input, "-", "_")
}
