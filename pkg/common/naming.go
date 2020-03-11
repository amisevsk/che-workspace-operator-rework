package common

import (
	"regexp"
	"strings"
)

var NonAlphaNumRegexp = regexp.MustCompile(`[^a-z0-9]+`)

func EndpointName(endpointName string) string {
	name := strings.ToLower(endpointName)
	name = NonAlphaNumRegexp.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	return name
}
