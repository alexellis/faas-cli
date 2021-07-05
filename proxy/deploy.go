// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/openfaas/faas-cli/stack"

	types "github.com/openfaas/faas-provider/types"
)

var (
	defaultCommandTimeout = 60 * time.Second
)

// FunctionResourceRequest defines a request to set function resources
type FunctionResourceRequest struct {
	Limits   *stack.FunctionResources
	Requests *stack.FunctionResources
}

// DeployFunctionSpec defines the spec used when deploying a function
type DeployFunctionSpec struct {
	FProcess                string
	FunctionName            string
	Image                   string
	RegistryAuth            string
	Language                string
	Replace                 bool
	EnvVars                 map[string]string
	Network                 string
	Constraints             []string
	Update                  bool
	Secrets                 []string
	Labels                  map[string]string
	Annotations             map[string]string
	FunctionResourceRequest FunctionResourceRequest
	ReadOnlyRootFilesystem  bool
	TLSInsecure             bool
	Token                   string
	Namespace               string
}

func generateFuncStr(spec *DeployFunctionSpec) string {

	if len(spec.Namespace) > 0 {
		return fmt.Sprintf("%s.%s", spec.FunctionName, spec.Namespace)
	}
	return spec.FunctionName
}

type DeployResponse struct {
	Message       string
	RollingUpdate bool
	URL           string
}

// DeployFunction first tries to deploy a function and if it exists will then attempt
// a rolling update. Warnings are suppressed for the second API call (if required.)
func (c *Client) DeployFunction(context context.Context, spec *DeployFunctionSpec) (*DeployResponse, *http.Response, error) {

	rollingUpdateInfo := fmt.Sprintf("Function %s already exists, attempting rolling-update.", spec.FunctionName)
	res, err := c.deploy(context, spec, spec.Update)

	if err != nil && IsUnknown(err) {
		return nil, res, err
	}

	var deployResponse DeployResponse
	if err == nil {
		deployResponse.Message = fmt.Sprintf("Deployed. %s.", res.Status)
		deployResponse.URL = fmt.Sprintf("%s/function/%s", c.GatewayURL.String(), generateFuncStr(spec))
	}

	if spec.Update == true && IsNotFound(err) {
		// Re-run the function with update=false
		res, err = c.deploy(context, spec, false)
		if err == nil {
			deployResponse.Message = fmt.Sprintf("Deployed. %s.", res.Status)
			deployResponse.URL = fmt.Sprintf("%s/function/%s", c.GatewayURL.String(), generateFuncStr(spec))
		}

	} else if res.StatusCode == http.StatusOK {
		deployResponse.Message += rollingUpdateInfo
		deployResponse.RollingUpdate = true

	}

	return &deployResponse, res, err
}

// deploy a function to an OpenFaaS gateway over REST
func (c *Client) deploy(context context.Context, spec *DeployFunctionSpec, update bool) (*http.Response, error) {
	// Need to alter Gateway to allow nil/empty string as fprocess, to avoid this repetition.
	var fprocessTemplate string
	if len(spec.FProcess) > 0 {
		fprocessTemplate = spec.FProcess
	}

	if spec.Replace {
		c.DeleteFunction(context, spec.FunctionName, spec.Namespace)
	}

	req := types.FunctionDeployment{
		EnvProcess:             fprocessTemplate,
		Image:                  spec.Image,
		Service:                spec.FunctionName,
		EnvVars:                spec.EnvVars,
		Constraints:            spec.Constraints,
		Secrets:                spec.Secrets,
		Labels:                 &spec.Labels,
		Annotations:            &spec.Annotations,
		ReadOnlyRootFilesystem: spec.ReadOnlyRootFilesystem,
		Namespace:              spec.Namespace,
	}

	hasLimits := false
	req.Limits = &types.FunctionResources{}
	if spec.FunctionResourceRequest.Limits != nil && len(spec.FunctionResourceRequest.Limits.Memory) > 0 {
		hasLimits = true
		req.Limits.Memory = spec.FunctionResourceRequest.Limits.Memory
	}
	if spec.FunctionResourceRequest.Limits != nil && len(spec.FunctionResourceRequest.Limits.CPU) > 0 {
		hasLimits = true
		req.Limits.CPU = spec.FunctionResourceRequest.Limits.CPU
	}
	if !hasLimits {
		req.Limits = nil
	}

	hasRequests := false
	req.Requests = &types.FunctionResources{}
	if spec.FunctionResourceRequest.Requests != nil && len(spec.FunctionResourceRequest.Requests.Memory) > 0 {
		hasRequests = true
		req.Requests.Memory = spec.FunctionResourceRequest.Requests.Memory
	}
	if spec.FunctionResourceRequest.Requests != nil && len(spec.FunctionResourceRequest.Requests.CPU) > 0 {
		hasRequests = true
		req.Requests.CPU = spec.FunctionResourceRequest.Requests.CPU
	}

	if !hasRequests {
		req.Requests = nil
	}

	reqBytes, _ := json.Marshal(&req)
	reader := bytes.NewReader(reqBytes)
	var request *http.Request

	method := http.MethodPost
	// "application/json"
	if update {
		method = http.MethodPut
	}

	var err error
	request, err = c.newRequest(method, "/system/functions", reader)

	if err != nil {
		return nil, err
	}

	res, err := c.doRequest(context, request)

	if err != nil {
		errMessage := fmt.Sprintln("Is OpenFaaS deployed? Do you need to specify the --gateway flag?")
		errMessage += fmt.Sprintln(err)
		return res, NewUnknown(errMessage, 0)
	}

	err = checkForAPIError(res)
	if err != nil {
		return res, err
	}
	return res, nil
}
