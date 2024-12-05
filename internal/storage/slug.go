package storage

import (
	"regexp"
	"strings"
)

func slugify(value string) string {
	return strings.ToLower(
		regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(value, "-"),
	)
}
