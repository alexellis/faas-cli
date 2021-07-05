// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package proxy

import (
	"context"
	"net/http"

	"testing"

	"regexp"

	"github.com/openfaas/faas-cli/test"
)

const tlsNoVerify = true

type deployProxyTest struct {
	title               string
	mockServerResponses []int
	replace             bool
	update              bool
	expectedMessage     string
	statusCode          int
}

func runDeployProxyTest(t *testing.T, deployTest deployProxyTest) {
	s := test.MockHttpServerStatus(
		t,
		deployTest.mockServerResponses...,
	)
	defer s.Close()

	cliAuth := NewTestAuth(nil)
	proxyClient, _ := NewClient(cliAuth, s.URL, nil, &defaultCommandTimeout)

	dRes, httpRes, _ := proxyClient.DeployFunction(context.TODO(), &DeployFunctionSpec{
		"fprocess",
		"function",
		"image",
		"dXNlcjpwYXNzd29yZA==",
		"language",
		deployTest.replace,
		nil,
		"network",
		[]string{},
		deployTest.update,
		[]string{},
		map[string]string{},
		map[string]string{},
		FunctionResourceRequest{},
		false,
		tlsNoVerify,
		"",
		"",
	})

	if httpRes.StatusCode != deployTest.statusCode {
		t.Fatalf("StatuCode did not match. expected: %d, got: %d", deployTest.statusCode, httpRes.StatusCode)
	}

	r := regexp.MustCompile(deployTest.expectedMessage)
	if !r.MatchString(dRes.Message) {
		t.Fatalf("Output not matched: %s", dRes.Message)
	}
}

func Test_RunDeployProxyTests(t *testing.T) {
	var deployProxyTests = []deployProxyTest{
		{
			title:               "200_Deploy",
			mockServerResponses: []int{http.StatusOK, http.StatusOK},
			replace:             true,
			update:              false,
			statusCode:          http.StatusOK,
			expectedMessage:     `(?m:Deployed)`,
		},
		{
			title:               "404_Deploy",
			mockServerResponses: []int{http.StatusOK, http.StatusNotFound},
			replace:             true,
			update:              false,
			statusCode:          http.StatusNotFound,
			expectedMessage:     "",
		},
		{
			title:               "UpdateFailedDeployed",
			mockServerResponses: []int{http.StatusNotFound, http.StatusOK},
			replace:             false,
			update:              true,
			statusCode:          http.StatusOK,
			expectedMessage:     `(?m:Deployed)`,
		},
	}
	for _, tst := range deployProxyTests {
		t.Run(tst.title, func(t *testing.T) {
			runDeployProxyTest(t, tst)
		})
	}
}

func Test_DeployFunction_generateFuncStr(t *testing.T) {

	testCases := []struct {
		name        string
		spec        *DeployFunctionSpec
		expectedStr string
		shouldErr   bool
	}{
		{
			name: "No Namespace",
			spec: &DeployFunctionSpec{
				"fprocess",
				"funcName",
				"image",
				"dXNlcjpwYXNzd29yZA==",
				"language",
				false,
				nil,
				"network",
				[]string{},
				false,
				[]string{},
				map[string]string{},
				map[string]string{},
				FunctionResourceRequest{},
				false,
				tlsNoVerify,
				"",
				"",
			},
			expectedStr: "funcName",
		},
		{name: "With Namespace",
			spec: &DeployFunctionSpec{
				"fprocess",
				"funcName",
				"image",
				"dXNlcjpwYXNzd29yZA==",
				"language",
				false,
				nil,
				"network",
				[]string{},
				false,
				[]string{},
				map[string]string{},
				map[string]string{},
				FunctionResourceRequest{},
				false,
				tlsNoVerify,
				"",
				"nameSpace",
			},
			expectedStr: "funcName.nameSpace",
		},
	}

	for _, testCase := range testCases {
		funcStr := generateFuncStr(testCase.spec)

		if funcStr != testCase.expectedStr {
			t.Fatalf("generateFuncStr %s\nwant: %s, got: %s", testCase.name, testCase.expectedStr, funcStr)
		}
	}
}

type testAuth struct {
	err error
}

func (c *testAuth) Set(req *http.Request) error {
	return c.err
}

func NewTestAuth(err error) ClientAuth {
	return &testAuth{
		err: err,
	}
}
