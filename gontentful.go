package gontentful

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	timeout = 10 * time.Second
	cdn     = "cdn.contentful.com"
	api     = "api.contentful.com"

	pathSpaces  = "/spaces/%s"
	pathEntries = pathSpaces + "/entries"

	headerContentfulContentType = "X-Contentful-Content-Type"
)

type Client struct {
	client       *http.Client
	headers      map[string]string
	Options      *ClientOptions
	AfterRequest func(c *Client, req *http.Request, res *http.Response, elapsed time.Duration)

	common  service
	Entries *EntriesService
	Spaces  *SpacesService
}

type service struct {
	client *Client
}

type ClientOptions struct {
	OrgID            string
	SpaceID          string
	ApiHost          string
	ApiToken         string
}

func NewClient(options *ClientOptions) *Client {
	client := &Client{
		Options: options,
		client: &http.Client{
			Timeout: timeout,
		},
		headers: map[string]string{
			"X-Contentful-Organization": options.OrgID,
			"Authorization":             fmt.Sprintf("Bearer %s", options.ApiToken),
			"Content-Type":              "application/vnd.contentful.delivery.v1+json",
		},
	}

	client.common.client = client
	client.Entries = (*EntriesService)(&client.common)
	client.Spaces = (*SpacesService)(&client.common)

	return client
}

func (c *Client) post(path string, body io.Reader) ([]byte, error) {
	return c.req(http.MethodPost, path, nil, body)
}

func (c *Client) get(path string, query url.Values) ([]byte, error) {
	return c.req(http.MethodGet, path, query, nil)
}

func (c *Client) req(method string, path string, query url.Values, body io.Reader) ([]byte, error) {
	host := ""
	if method == http.MethodGet {
		host = cdn
	} else if method == http.MethodPost {
		host = api
	}

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

	// set headers
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	start := time.Now()
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if c.AfterRequest != nil {
		c.AfterRequest(c, req, res, time.Since(start))
	}

	if res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusBadRequest {
		return ioutil.ReadAll(res.Body)
	}

	apiError := parseError(req, res)

	// return apiError if it is not rate limit error
	if _, ok := apiError.(RateLimitExceededError); !ok {
		return nil, apiError
	}

	resetHeader := res.Header.Get("x-contentful-ratelimit-reset")

	// return apiError if Ratelimit-Reset header is not presented
	if resetHeader == "" {
		return nil, apiError
	}

	// wait X-Contentful-Ratelimit-Reset amount of seconds
	waitSeconds, err := strconv.Atoi(resetHeader)
	if err != nil {
		return nil, apiError
	}

	// retry on rate limit
	time.Sleep(time.Second * time.Duration(waitSeconds))
	return c.do(req)
}
