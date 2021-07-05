// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package proxy

import (
	"context"

	"fmt"
	"net/http"

	types "github.com/openfaas/faas-provider/types"
)

// ListFunctions list deployed functions
func (c *Client) ListFunctions(ctx context.Context, namespace string) ([]types.FunctionStatus, *http.Response, error) {
	var (
		results      []types.FunctionStatus
		listEndpoint string
		err          error
	)

	c.AddCheckRedirect(func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	})

	listEndpoint = systemPath
	if len(namespace) > 0 {
		listEndpoint, err = addQueryParams(listEndpoint, map[string]string{namespaceKey: namespace})
		if err != nil {
			return results, nil, err
		}
	}

	getRequest, err := c.newRequest(http.MethodGet, listEndpoint, nil)
	if err != nil {
		return results, nil, fmt.Errorf("cannot connect to OpenFaaS on URL: %s", c.GatewayURL.String())
	}

	res, err := c.doRequest(ctx, getRequest)
	if err != nil {
		return results, res, fmt.Errorf("cannot connect to OpenFaaS on URL: %s", c.GatewayURL.String())
	}

	err = checkForAPIError(res)
	if err != nil {
		return results, res, err
	}

	err = parseResponse(res, &results)
	if err != nil {
		return results, res, err
	}

	return results, res, nil
}
