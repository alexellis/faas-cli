package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	types "github.com/openfaas/faas-provider/types"
)

const (
	secretEndpoint = "/system/secrets"
)

// GetSecretList get secrets list
func (c *Client) GetSecretList(ctx context.Context, namespace string) ([]types.Secret, *http.Response, error) {
	var (
		results    []types.Secret
		err        error
		secretPath = secretEndpoint
	)

	if len(namespace) > 0 {
		secretPath, err = addQueryParams(secretPath, map[string]string{namespaceKey: namespace})
	}

	getRequest, err := c.newRequest(http.MethodGet, secretPath, nil)

	if err != nil {
		return nil, nil, fmt.Errorf("cannot connect to OpenFaaS on URL: %s", c.GatewayURL.String())
	}

	res, err := c.doRequest(ctx, getRequest)
	if err != nil {
		return nil, res, fmt.Errorf("cannot connect to OpenFaaS on URL: %s", c.GatewayURL.String())
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	bytesOut, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, res, err
	}

	switch res.StatusCode {
	case http.StatusOK, http.StatusAccepted:
		jsonErr := json.Unmarshal(bytesOut, &results)
		if jsonErr != nil {
			return nil, res, fmt.Errorf("cannot parse result from OpenFaaS: %s", jsonErr.Error())
		}

	default:
		return nil, res, NewOpenFaaSError(string(bytesOut), res.StatusCode)
	}

	return results, res, nil
}

// UpdateSecret update a secret via the OpenFaaS API by name
func (c *Client) UpdateSecret(ctx context.Context, secret types.Secret) (*http.Response, error) {
	reqBytes, _ := json.Marshal(&secret)

	putRequest, err := c.newRequest(http.MethodPut, secretEndpoint, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, err
	}

	res, err := c.doRequest(ctx, putRequest)
	if err != nil {
		return res, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}
	bytesOut, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return res, err
	}

	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusAccepted {
		return res, nil
	}

	return res, NewOpenFaaSError(string(bytesOut), res.StatusCode)
}

// RemoveSecret remove a secret via the OpenFaaS API by name
func (c *Client) RemoveSecret(ctx context.Context, secret types.Secret) (*http.Response, error) {
	body, _ := json.Marshal(secret)
	req, err := c.newRequest(http.MethodDelete, secretEndpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	res, err := c.doRequest(ctx, req)
	if err != nil {
		return res, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusAccepted {
		return res, nil
	}

	bytesOut, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return res, err
	}

	return res, NewOpenFaaSError(string(bytesOut), res.StatusCode)
}

// CreateSecret create secret
func (c *Client) CreateSecret(ctx context.Context, secret types.Secret) (*http.Response, error) {
	reqBytes, _ := json.Marshal(&secret)
	reader := bytes.NewReader(reqBytes)

	req, err := c.newRequest(http.MethodPost, secretEndpoint, reader)

	if err != nil {
		return nil, err
	}

	res, err := c.doRequest(ctx, req)
	if err != nil {
		return res, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusAccepted || res.StatusCode == http.StatusCreated {
		return res, nil
	}

	bytesOut, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return res, err
	}

	return res, NewOpenFaaSError(string(bytesOut), res.StatusCode)
}
