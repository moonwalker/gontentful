package gontentful

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gosimple/slug"
	"github.com/moonwalker/moonbase/pkg/content"
	gh "github.com/moonwalker/moonbase/pkg/github"
	"gopkg.in/yaml.v3"
)

func getAccessToken() string {
	ght := os.Getenv("GITHUB_TOKEN")
	if len(ght) == 0 {
		ght = os.Getenv("GH_TOKEN")
	}
	return ght
}

func getConfig(ctx context.Context, owner string, repo string, ref string) *Config {
	accessToken := getAccessToken()
	data, _, _ := gh.GetBlob(ctx, accessToken, owner, repo, ref, configPath)
	return parseConfig(data, accessToken)
}

func parseConfig(data []byte, token string) *Config {
	cfg := &Config{
		Token: token,
	}

	err := yaml.Unmarshal(data, cfg)
	if err != nil {
		json.Unmarshal(data, cfg)
	}

	return cfg
}

func mergeMaps[M ~map[K]V, K comparable, V any](dst M, src M) {
	for k, v := range src {
		dst[k] = v
	}
}

func extractContenttype(ct string, path string, idx int) string {
	if ct != "" {
		return ct
	}
	dirs := strings.Split(filepath.Dir(path), "/")
	return dirs[len(dirs)-idx]
}

func extractFileInfo(fn string) (string, string) {
	ext := filepath.Ext(fn)
	locale := strings.TrimSuffix(fn, ext)
	return locale, ext
}

func extractLocale(path string, ext string) string {
	return strings.TrimSuffix(filepath.Base(path), ext)
}

func GetImageFileName(fileName string, sysId string, locale string) string {
	ext := filepath.Ext(fileName)
	return slug.Make(fmt.Sprintf("%s_%s-%s", fileName[:len(fileName)-len(ext)], sysId, locale)) + ext
}

func getDefaultLocale(locales []*Locale) string {
	hasDefault := false
	for _, loc := range locales {
		code := strings.ToLower(loc.Code)
		if loc.Default {
			return code
		}
		if code == DefaultLocale {
			hasDefault = true
		}
	}
	if len(locales) > 0 && !hasDefault {
		return strings.ToLower(locales[0].Code)
	}

	return DefaultLocale
}

func validateField(field string, value interface{}, validations []*content.Validation) error {
	for _, val := range validations {
		switch val.Type {
		case "required":
			s, ok := value.(string)
			if val.Value.(bool) && (value == nil || (ok && s == "")) {
				return fmt.Errorf("required validation failed on %s", field)
			}
		case "in":
			vals := val.Value.([]interface{})
			found := false
			for _, v := range vals {
				if v == value {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("in validation failed on %s for %v", field, value)
			}

		case "size":
			m, _ := val.Value.(map[string]interface{})
			j := value.(float64)
			for k, v := range m {
				i := v.(float64)
				if k == "min" {
					if i < j {
						return fmt.Errorf("min validation failed on %s for %v", field, value)
					}
				}
				if k == "max" {
					i := v.(float64)
					if i < j {
						return fmt.Errorf("max validation failed on %s for %v", field, value)
					}
				}
			}
		case "regexp":
			m, _ := val.Value.(map[string]interface{})
			regStr := ""
			for k, v := range m {
				if k == "pattern" {
					regStr = regStr + v.(string)
				}
				if k == "flags" {
					regStr = v.(string) + regStr
				}
			}
			reg, err := regexp.Compile(regStr)
			if err != nil {
				return err
			}
			if !reg.MatchString(value.(string)) {
				return fmt.Errorf("regexp validation failed on %s for %v", field, value)
			}
		}
	}
	return nil
}
