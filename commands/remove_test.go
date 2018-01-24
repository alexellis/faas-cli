// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package commands

import (
	"net/http"
	"testing"

	"github.com/openfaas/faas-cli/test"
)

func Test_remove(t *testing.T) {
	s := test.MockHttpServer(t, []test.Request{
		{
			Method:             http.MethodDelete,
			Uri:                "/system/functions",
			ResponseStatusCode: http.StatusOK,
		},
	})
	defer s.Close()

	resetForTest()

	removeCmd := newRemoveCmd()
	removeCmd.SetArgs([]string{
		"--gateway=" + s.URL,
		"test-function",
	})
	if err := removeCmd.Execute(); err != nil {
		t.Fatalf("Inexpected error: %s", err)
	}
}
