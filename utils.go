package gontentful

import (
	"fmt"
	"regexp"
	"strings"
)

func fmtLocale(code string) string {
	return strings.ToLower(code)
}

func fmtTableName(contentType string, locale string) string {
	return fmt.Sprintf("%s$%s", strings.ToLower(contentType), fmtLocale(locale))
}

func fmtTablePublishName(contentType string, locale string) string {
	return fmt.Sprintf("%s$%s$publish", strings.ToLower(contentType), fmtLocale(locale))
}

var camel = regexp.MustCompile("(^[^A-Z]*|[A-Z]*)([A-Z][^A-Z]+|$)")
var snake = regexp.MustCompile(`([_ ]\w)`)

func toSnakeCase(s string) string {
	var a []string
	for _, sub := range camel.FindAllStringSubmatch(s, -1) {
		if sub[1] != "" {
			a = append(a, sub[1])
		}
		if sub[2] != "" {
			a = append(a, sub[2])
		}
	}
	return strings.ToLower(strings.Join(a, "_"))
}

func toCamelCase(s string) string {
	return snake.ReplaceAllStringFunc(s, func(w string) string {
		return strings.ToUpper(string(w[1]))
	})
}
