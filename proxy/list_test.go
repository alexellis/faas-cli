// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package proxy

import (
	"context"
	"net/http"
	"reflect"

	"testing"

	"github.com/openfaas/faas-cli/test"
	types "github.com/openfaas/faas-provider/types"
)

var wantListFunctionsResponse = []types.FunctionStatus{
	{
		Name:            "func-test1",
		Image:           "image-test1",
		Replicas:        1,
		InvocationCount: 1,
		EnvProcess:      "env-process test1",
	},
	{
		Name:            "func-test2",
		Image:           "image-test2",
		Replicas:        2,
		InvocationCount: 2,
		EnvProcess:      "env-process test2",
	},
}

func Test_ListFunctions(t *testing.T) {

	s := test.MockHttpServer(t, []test.Request{
		{
			ResponseStatusCode: http.StatusOK,
			ResponseBody:       wantListFunctionsResponse,
		},
	})
	defer s.Close()

	cliAuth := NewTestAuth(nil)
	client, _ := NewClient(cliAuth, s.URL, nil, &defaultCommandTimeout)
	result, _, err := client.ListFunctions(context.Background(), "")

	if err != nil {
		t.Fatalf("Error returned: %s", err)
	}
	for k, v := range result {
		if !reflect.DeepEqual(wantListFunctionsResponse[k], v) {
			t.Fatalf("Want: %#v - Got: %#v", wantListFunctionsResponse[k], v)
		}
	}
}

func Test_ListFunctions_Not200(t *testing.T) {
	s := test.MockHttpServerStatus(t, http.StatusBadRequest)

	cliAuth := NewTestAuth(nil)
	client, _ := NewClient(cliAuth, s.URL, nil, &defaultCommandTimeout)
	_, _, err := client.ListFunctions(context.Background(), "")

	if err == nil {
		t.Fatalf("Error was not returned")
	}

	if !IsBadRequest(err) {
		t.Fatalf("Error not matched: %s", err)
	}
}
