package users

import (
	"regexp"
	"strings"
)

// Normalize lowercases and safens a username.
func Normalize(username string) string {
	username = strings.ToLower(username)

	// Strip special characters.
	re := regexp.MustCompile(`[./\\]+`)
	return re.ReplaceAllString(username, "")
}
