// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package proxy

import (
	"net/http"

	"testing"

	"regexp"

	"github.com/openfaas/faas-cli/test"
)

func Test_DeleteFunction(t *testing.T) {
	s := test.MockHttpServerStatus(t, http.StatusOK)
	defer s.Close()

	stdout := test.CaptureStdout(func() {
		DeleteFunction(s.URL, "function-to-delete")
	})

	r := regexp.MustCompile(`(?m:Removing old function.)`)
	if !r.MatchString(stdout) {
		t.Fatalf("Want: %s, got: %s", "Removing old function", stdout)
	}
}

func Test_DeleteFunction_404(t *testing.T) {
	s := test.MockHttpServerStatus(t, http.StatusNotFound)
	defer s.Close()

	stdout := test.CaptureStdout(func() {
		DeleteFunction(s.URL, "function-to-delete")
	})

	r := regexp.MustCompile(`(?m:No existing function to remove)`)
	if !r.MatchString(stdout) {
		t.Fatalf("Want: %s, got: %s", "No existing function to remove", stdout)
	}
}

func Test_DeleteFunction_Not2xxAnd404(t *testing.T) {
	s := test.MockHttpServerStatus(t, http.StatusInternalServerError)
	defer s.Close()

	stdout := test.CaptureStdout(func() {
		DeleteFunction(s.URL, "function-to-delete")
	})

	r := regexp.MustCompile(`(?m:Server returned unexpected status code)`)
	if !r.MatchString(stdout) {
		t.Fatalf("Output not matched: %s", stdout)
	}
}

func Test_DeleteFunction_BadURL(t *testing.T) {
	url := "127.0.0.1:8080"

	stdout := test.CaptureStdout(func() {
		DeleteFunction(url, "function-to-delete")
	})

	r := regexp.MustCompile(`(?m:first path segment in URL cannot contain colon)`)
	if !r.MatchString(stdout) {
		t.Fatalf("Output not matched: %s", stdout)
	}
}
