package proxy

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/openfaas/faas-cli/test"
	"github.com/openfaas/faas-provider/types"
)

func TestSecretList(t *testing.T) {
	type testCases struct {
		Name             string
		MockResponseCode int
		ExpectedError    error
		ExpectedResponse interface{}
	}

	tests := []testCases{
		{
			Name:             "Expect 200OK with list",
			MockResponseCode: http.StatusOK,
			ExpectedError:    nil,
			ExpectedResponse: makeSecretList([]string{"one", "two", "three"}),
		},
		{
			Name:             "Expect 200OK empty secrets",
			MockResponseCode: http.StatusOK,
			ExpectedError:    nil,
			ExpectedResponse: []string{},
		},
		{
			Name:             "Expect 200OK with invalid response",
			MockResponseCode: http.StatusOK,
			ExpectedError:    fmt.Errorf("cannot parse result from OpenFaaS: invalid character 'T' looking for beginning of value"),
			ExpectedResponse: "This is not ok",
		},
		{
			Name:             "Expect 202",
			MockResponseCode: http.StatusAccepted,
			ExpectedError:    nil,
			ExpectedResponse: makeSecretList([]string{"one", "two", "three"}),
		},
		{
			Name:             "Expect 400",
			MockResponseCode: http.StatusBadRequest,
			ExpectedError: &OpenFaaSError{
				StatusCode: 400,
				Message:    "Bad Request",
			},
			ExpectedResponse: "Bad Request",
		},
		{
			Name:             "Expect 500",
			MockResponseCode: http.StatusInternalServerError,
			ExpectedError: &OpenFaaSError{
				StatusCode: 500,
				Message:    "Internal server Error",
			},
			ExpectedResponse: "Internal server Error",
		},
		{
			Name:             "Expect 401",
			MockResponseCode: http.StatusUnauthorized,
			ExpectedError: &OpenFaaSError{
				StatusCode: 401,
				Message:    "Not Authorized",
			},
			ExpectedResponse: "Not Authorized",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.Name, func(t *testing.T) {
			s := test.MockHttpServer(t, []test.Request{
				{
					ResponseStatusCode: testCase.MockResponseCode,
					ResponseBody:       testCase.ExpectedResponse,
				},
			})
			defer s.Close()
			client, _ := NewClient(NewTestAuth(nil), s.URL, nil, nil)

			secrets, _, err := client.GetSecretList(context.Background(), "openfaas-fn")

			if testCase.ExpectedError == nil && err != nil {
				t.Errorf("Error returned: %v", err)
			}

			if testCase.ExpectedError != nil {
				if err.Error() != testCase.ExpectedError.Error() {
					t.Fatalf("Expected %v, got %v", testCase.ExpectedError, err)
				}
			}

			switch testCase.ExpectedResponse.(type) {
			case []string:
				expected := makeSecretList(testCase.ExpectedResponse.([]string))
				for k, v := range secrets {
					if expected[k] != v {
						t.Fatalf("Expeceted: %#v - Actual: %#v", wantListFunctionsResponse[k], v)
					}
				}

			}
		})
	}
}

func makeSecretList(i []string) []types.Secret {
	var secrets []types.Secret
	for _, s := range i {
		secrets = append(secrets, types.Secret{Name: s})
	}
	return secrets
}

func TestSecretCreate(t *testing.T) {
	type testCases struct {
		Name             string
		MockResponseCode int
		ExpectedError    error
		ExpectedResponse interface{}
		Namespace        string
	}

	tests := []testCases{
		{
			Name:             "Expect 200",
			MockResponseCode: http.StatusOK,
			ExpectedError:    nil,
		},
		{
			Name:             "Expect 201",
			MockResponseCode: http.StatusOK,
			ExpectedError:    nil,
		},
		{
			Name:             "Expect 202",
			MockResponseCode: http.StatusAccepted,
			ExpectedError:    nil,
		},
		{
			Name:             "Expect 400",
			MockResponseCode: http.StatusBadRequest,
			ExpectedError: &OpenFaaSError{
				StatusCode: http.StatusBadRequest,
				Message:    "Bad Request",
			},
			ExpectedResponse: "Bad Request",
		},
		{
			Name:             "Expect conflict",
			MockResponseCode: http.StatusConflict,
			ExpectedError: OpenFaaSError{
				StatusCode: http.StatusConflict,
				Message:    "Conflict",
			},
			ExpectedResponse: "Conflict",
		},
		{
			Name:             "Expect 500",
			MockResponseCode: http.StatusInternalServerError,
			ExpectedError: &OpenFaaSError{
				StatusCode: http.StatusInternalServerError,
				Message:    "Internal server Error",
			},
			ExpectedResponse: "Internal server Error",
		},
		{
			Name:             "Expect 401",
			MockResponseCode: http.StatusUnauthorized,
			ExpectedError: &OpenFaaSError{
				StatusCode: http.StatusUnauthorized,
				Message:    "Not Authorized",
			},
			ExpectedResponse: "Not Authorized",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.Name, func(t *testing.T) {
			s := test.MockHttpServer(t, []test.Request{
				{
					ResponseStatusCode: testCase.MockResponseCode,
					ResponseBody:       testCase.ExpectedResponse,
				},
			})
			defer s.Close()
			client, _ := NewClient(NewTestAuth(nil), s.URL, nil, nil)

			sec := types.Secret{
				Name:      testCase.Name,
				Namespace: testCase.Namespace,
				Value:     "",
			}
			_, err := client.CreateSecret(context.Background(), sec)

			if testCase.ExpectedError == nil && err != nil {
				t.Errorf("Error returned: %v", err)
			}

			if testCase.ExpectedError != nil {
				if err.Error() != testCase.ExpectedError.Error() {
					t.Fatalf("Expected %v, got %v", testCase.ExpectedError, err)
				}
			}

		})
	}
}

func TestSecretDelete(t *testing.T) {
	type testCases struct {
		Name             string
		MockResponseCode int
		ExpectedError    error
		ExpectedResponse interface{}
	}

	tests := []testCases{
		{
			Name:             "Expect 200",
			MockResponseCode: http.StatusOK,
			ExpectedError:    nil,
		},
		{
			Name:             "Expect 202",
			MockResponseCode: http.StatusAccepted,
			ExpectedError:    nil,
		},
		{
			Name:             "Expect 400",
			MockResponseCode: http.StatusBadRequest,
			ExpectedError: &OpenFaaSError{
				StatusCode: http.StatusBadRequest,
				Message:    "Bad Request",
			},
			ExpectedResponse: "Bad Request",
		},
		{
			Name:             "Expect conflict",
			MockResponseCode: http.StatusConflict,
			ExpectedError: OpenFaaSError{
				StatusCode: http.StatusConflict,
				Message:    "Conflict",
			},
			ExpectedResponse: "Conflict",
		},
		{
			Name:             "Expect 500",
			MockResponseCode: http.StatusInternalServerError,
			ExpectedError: &OpenFaaSError{
				StatusCode: http.StatusInternalServerError,
				Message:    "Internal server Error",
			},
			ExpectedResponse: "Internal server Error",
		},
		{
			Name:             "Expect 401",
			MockResponseCode: http.StatusUnauthorized,
			ExpectedError: &OpenFaaSError{
				StatusCode: http.StatusUnauthorized,
				Message:    "Not Authorized",
			},
			ExpectedResponse: "Not Authorized",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.Name, func(t *testing.T) {
			s := test.MockHttpServer(t, []test.Request{
				{
					ResponseStatusCode: testCase.MockResponseCode,
					ResponseBody:       testCase.ExpectedResponse,
				},
			})
			defer s.Close()
			client, _ := NewClient(NewTestAuth(nil), s.URL, nil, nil)

			sec := types.Secret{
				Name:  testCase.Name,
				Value: "",
			}
			_, err := client.RemoveSecret(context.Background(), sec)

			if testCase.ExpectedError == nil && err != nil {
				t.Errorf("Error returned: %v", err)
			}

			if testCase.ExpectedError != nil {
				if err.Error() != testCase.ExpectedError.Error() {
					t.Fatalf("Expected %v, got %v", testCase.ExpectedError, err)
				}
			}

		})
	}
}
