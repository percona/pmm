package enums

import (
	"fmt"
	"strings"
)

// ConvertEnum converts flag value to value supported by API.
func ConvertEnum(prefix string, value string) string {
	if value == "" {
		return ""
	}
	return fmt.Sprintf("%s_%s", prefix, strings.ToUpper(value))
}
