package gontentful

import (
	"encoding/json"
)

type Sys struct {
	ID          string       `json:"id"`
	Type        string       `json:"type,omitempty"`
	LinkType    string       `json:"linkType,omitempty"`
	ContentType *ContentType `json:"contentType,omitempty"`
}

type ContentType struct {
	Sys *Sys `json:"sys"`
}

type Entries struct {
	Sys      *Sys    `json:"sys"`
	Total    int     `json:"total"`
	Skip     int     `json:"skip"`
	Limit    int     `json:"limit"`
	Items    []Entry `json:"items"`
	Includes Include `json:"includes,omitempty"`
}

type Include struct {
	Entry []Entry `json:"entry,omitempty"`
	Asset []Entry `json:"asset,omitempty"`
}

type Entry struct {
	Sys    *Sys                   `json:"sys"`
	Locale string                 `json:"locale,omitempty"`
	Fields map[string]interface{} `json:"fields"`
}

type Space struct {
	Sys     *Sys     `json:"sys"`
	Name    string   `json:"name"`
	Locales []Locale `json:"locales"`
}

type Locale struct {
	Code         string `json:"code"`
	Default      bool   `json:"default"`
	Name         string `json:"name"`
	FallbackCode string `json:"fallbackCode"`
}

type errorResponse struct {
	Message string `json:"message,omitempty"`
}

func UnmarshalEntries(body []byte) (entries *Entries, err error) {
	entries = &Entries{}
	err = json.Unmarshal(body, entries)
	return
}
