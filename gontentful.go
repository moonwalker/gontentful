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
	timeout    = 10 * time.Second
	urlCdn     = "cf-cdn.mw.zone"
	urlCma     = "api.contentful.com"
	urlPreview = "preview.contentful.com"

	pathSpaces         = "/spaces/%s"
	pathEntries        = pathSpaces + "/entries"
	pathEntry          = pathEntries + "/%s"
	pathEntriesPublish = pathEntry + "/published"
	pathEntriesArchive = pathEntry + "/archived"
	pathSync           = "/spaces/%s/sync"

	headerContentfulContentType  = "X-Contentful-Content-Type"
	headerContentulVersion       = "X-Contentful-Version"
	headerContentfulOrganization = "X-Contentful-Organization"
	headerContentType            = "Content-Type"
	headerAuthorization          = "Authorization"
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
	OrgID        string
	SpaceID      string
	ApiToken     string
	PreviewToken string
	CMAToken     string
}

func NewClient(options *ClientOptions) *Client {
	client := &Client{
		Options: options,
		client: &http.Client{
			Timeout: timeout,
		},
		headers: map[string]string{
			headerContentfulOrganization: options.OrgID,
			headerContentType:            "application/vnd.contentful.delivery.v1+json",
		},
	}

	client.common.client = client
	client.Entries = (*EntriesService)(&client.common)
	client.Spaces = (*SpacesService)(&client.common)

	return client
}

func (c *Client) get(path string, query url.Values, preview bool) ([]byte, error) {
	return c.req(http.MethodGet, path, query, nil, preview)
}

func (c *Client) post(path string, body io.Reader) ([]byte, error) {
	return c.req(http.MethodPost, path, nil, body, false)
}

func (c *Client) put(path string, body io.Reader) ([]byte, error) {
	return c.req(http.MethodPut, path, nil, body, false)
}

func (c *Client) delete(path string) ([]byte, error) {
	return c.req(http.MethodDelete, path, nil, nil, false)
}

func (c *Client) req(method string, path string, query url.Values, body io.Reader, preview bool) ([]byte, error) {
	// Set correct host and token for the request
	host := ""
	authToken := ""
	if method == http.MethodGet {
		if preview {
			host = urlPreview
			authToken = fmt.Sprintf("Bearer %s", c.Options.PreviewToken)
		} else {
			host = urlCdn
			authToken = fmt.Sprintf("Bearer %s", c.Options.ApiToken)
		}
	} else {
		host = urlCma
		authToken = fmt.Sprintf("Bearer %s", c.Options.CMAToken)
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
		if value != "" {
			req.Header.Set(key, value)
		}
	}
	// add auth header
	req.Header.Set(headerAuthorization, authToken)

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
