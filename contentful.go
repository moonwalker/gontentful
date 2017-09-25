package contentful

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultTimeout     = 10 * time.Second
	defaultAPIHostname = "cdn.contentful.com"
)

type ContentfulClient struct {
	client  *http.Client
	headers map[string]string
}

// types

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

type ErrorResponse struct {
	Message string `json:"message,omitempty"`
}

func NewContentfulClient() *ContentfulClient {
	return &ContentfulClient{
		client: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (c *ContentfulClient) GetSpace(params *map[string]interface{}) (*Space, error) {
	query := url.Values{}
	for key, value := range *params {
		query.Set(key, fmt.Sprintf("%v", value))
	}
	path := fmt.Sprintf("/spaces/%s", query.Get("space_id"))

	var space Space
	err := c.get(path, query, &space)
	if err != nil {
		return nil, err
	}

	return &space, err
}

func (c *ContentfulClient) Entries(params *map[string]interface{}) (*Entries, error) {
	query := url.Values{}
	for key, value := range *params {
		query.Set(key, fmt.Sprintf("%v", value))
	}

	path := fmt.Sprintf("/spaces/%s/entries", query.Get("space_id"))

	var entries Entries
	err := c.get(path, query, &entries)
	if err != nil {
		return nil, err
	}

	return &entries, err
}

func (c *ContentfulClient) get(path string, query url.Values, res interface{}) error {
	return c.req(http.MethodGet, path, query, nil, res)
}

func (c *ContentfulClient) req(method string, path string, query url.Values, body io.Reader, v interface{}) error {
	apiHostname := query.Get("api_hostname")
	accessToken := query.Get("access_token")

	// cleanup
	query.Del("api_hostname")
	query.Del("access_token")
	query.Del("space_id")

	if apiHostname == "" {
		apiHostname = defaultAPIHostname
	}

	u := &url.URL{
		Scheme: "https",
		Host:   apiHostname,
		Path:   path,
	}
	u.RawQuery = query.Encode()

	fmt.Println(u.String())

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusBadRequest {
		if v != nil {
			defer res.Body.Close()
			err = json.NewDecoder(res.Body).Decode(v)
			if err != nil {
				return err
			}
		}

		return nil
	}

	var e ErrorResponse
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&e)
	if err != nil {
		return err
	}

	return errors.New(e.Message)
}
