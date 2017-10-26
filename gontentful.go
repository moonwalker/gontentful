package gontentful

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

const (
	timeout  = 10 * time.Second
	hostname = "cdn.contentful.com"

	pathSpace   = "/spaces/%s"
	pathEntries = pathSpace + "/entries"
)

type Client struct {
	client       *http.Client
	Options      *ClientOptions
	AfterRequest func(c *Client, req *http.Request, elapsed time.Duration)
}

type ClientOptions struct {
	ApiToken string
	SpaceID  string
	ApiHost  string
}

func NewClient(options *ClientOptions) *Client {
	return &Client{
		Options: options,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) GetSpace(query url.Values) ([]byte, error) {
	path := fmt.Sprintf(pathSpace, c.Options.SpaceID)
	return c.get(path, query)
}

func (c *Client) GetEntries(query url.Values) ([]byte, error) {
	path := fmt.Sprintf(pathEntries, c.Options.SpaceID)
	return c.get(path, query)
}

func (c *Client) get(path string, query url.Values) ([]byte, error) {
	return c.req(http.MethodGet, path, query, nil)
}

func (c *Client) req(method string, path string, query url.Values, body io.Reader) ([]byte, error) {
	host := hostname
	if c.Options.ApiHost != "" {
		host = c.Options.ApiHost
	}

	u := &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   path,
	}
	u.RawQuery = query.Encode()

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Options.ApiToken))

	start := time.Now()
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	c.AfterRequest(c, req, time.Since(start))

	if res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusBadRequest {
		return ioutil.ReadAll(res.Body)
	}

	var e errorResponse
	err = json.NewDecoder(res.Body).Decode(&e)
	if err != nil {
		return nil, err
	}

	return nil, errors.New(e.Message)
}
