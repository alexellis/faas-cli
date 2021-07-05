package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	gopath "path"
	"strings"
	"time"
)

//Client an API client to perform all operations
type Client struct {
	httpClient *http.Client
	//ClientAuth a type implementing ClientAuth interface for client authentication
	ClientAuth ClientAuth
	//GatewayURL base URL of OpenFaaS gateway
	GatewayURL *url.URL
	//UserAgent user agent for the client
	UserAgent string
}

//ClientAuth an interface for client authentication.
// to add authentication to the client implement this interface
type ClientAuth interface {
	Set(req *http.Request) error
}

//NewClient initializes a new API client
func NewClient(auth ClientAuth, gatewayURL string, transport http.RoundTripper, timeout *time.Duration) (*Client, error) {
	gatewayURL = strings.TrimRight(gatewayURL, "/")
	baseURL, err := url.Parse(gatewayURL)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway URL: %s", gatewayURL)
	}

	client := &http.Client{}
	if timeout != nil {
		client.Timeout = *timeout
	}

	if transport != nil {
		client.Transport = transport
	}

	return &Client{
		ClientAuth: auth,
		httpClient: client,
		GatewayURL: baseURL,
	}, nil
}

//newRequest create a new HTTP request with authentication
func (c *Client) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	// deep copy gateway url and then add the supplied path  and args to the copy so that
	// we preserve the original gateway URL as much as possible
	endpoint, err := url.Parse(c.GatewayURL.String())
	if err != nil {
		return nil, err
	}
	endpoint.Path = gopath.Join(endpoint.Path, u.Path)
	endpoint.RawQuery = u.RawQuery

	req, err := http.NewRequest(method, endpoint.String(), body)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	c.ClientAuth.Set(req)

	return req, err
}

//doRequest perform an HTTP request with context
func (c *Client) doRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)

	if val, ok := os.LookupEnv("OPENFAAS_DUMP_HTTP"); ok && val == "true" {
		dump, err := httputil.DumpRequest(req, true)
		if err != nil {
			return nil, err
		}
		fmt.Println(string(dump))
	}
	resp, err := c.httpClient.Do(req)

	if err != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	return resp, err
}

func addQueryParams(u string, params map[string]string) (string, error) {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return u, err
	}

	qs := parsedURL.Query()
	for key, value := range params {
		qs.Add(key, value)
	}
	parsedURL.RawQuery = qs.Encode()
	return parsedURL.String(), nil
}

//AddCheckRedirect add CheckRedirect to the client
func (c *Client) AddCheckRedirect(checkRedirect func(*http.Request, []*http.Request) error) {
	c.httpClient.CheckRedirect = checkRedirect
}

// parseResponse function parses HTTP response body into a typed struct
// or copies to io.Writer object. If it fails to decode response body
// then it re-construct it so that it can be read later
func parseResponse(res *http.Response, v interface{}) error {
	var err error
	defer res.Body.Close()

	switch v := v.(type) {
	case nil:
	case io.Writer:
		_, err = io.Copy(v, res.Body)
	default:
		data, err := ioutil.ReadAll(res.Body)
		if err == io.EOF {
			err = nil // ignore EOF errors caused by empty response body
		}

		decErr := json.Unmarshal(data, v)
		if decErr != nil {
			err = decErr
			// In case of JSON decode error, re-construct response body
			res.Body = ioutil.NopCloser(bytes.NewBuffer(data))
		}
	}
	return err
}
