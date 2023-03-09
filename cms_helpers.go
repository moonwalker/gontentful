package gontentful

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	gh "github.com/moonwalker/moonbase/pkg/github"
	"gopkg.in/yaml.v3"
)

func parseFileName(fn string) (string, string, error) {
	ext := filepath.Ext(fn)
	if ext != ".json" {
		return "", "", errors.New(fmt.Sprintf("incorrect file format: %s", ext))
	}

	s := strings.Split(strings.TrimSuffix(fn, ext), "_")
	if len(s) != 2 || len(s[0]) == 0 || len(s[1]) == 0 {
		return "", "", errors.New(fmt.Sprintf("incorrect filename: %s", fn))
	}

	return s[0], s[1], nil
}

func getConfig(ctx context.Context, accessToken string, owner string, repo string, ref string) *Config {
	data, _, _ := gh.GetBlob(ctx, accessToken, owner, repo, ref, configPath)
	return parseConfig(data)
}

func parseConfig(data []byte) *Config {
	cfg := &Config{}

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

func extractContentype(path string) string {
	items := strings.Split(path, "/")
	if len(items) > 1 {
		return items[len(items)-2]
	}
	return ""
}
