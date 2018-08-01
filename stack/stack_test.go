// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package stack

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

const TestData_1 string = `provider:
  name: faas
  gateway: http://127.0.0.1:8080
  network: "func_functions"

functions:
  url-ping:
    lang: python
    handler: ./sample/url-ping
    image: alexellis/faas-url-ping

  nodejs-echo:
    lang: node
    handler: ./sample/nodejs-echo
    image: alexellis/faas-nodejs-echo

  imagemagick:
    lang: dockerfile
    handler: ./sample/imagemagick
    image: functions/resizer
    fprocess: "convert - -resize 50% fd:1"

  ruby-echo:
    lang: ruby
    handler: ./sample/ruby-echo
    image: alexellis/ruby-echo

  abcd-eeee:
    lang: node
    handler: ./sample/abcd-eeee
    image: stuff2/stuff23423
`
const TestData_2 string = `provider:
  name: faas
  gateway: http://127.0.0.1:8080
  network: "func_functions"
`
const TestData_ExtResources string = `provider:
  name: faas
  gateway: http://127.0.0.1:8080

functions:
  f1:
    lang: node
    handler: handler
    image: image
    limits:
      cpu: 0.1
      vendor.domain/gpu: 1
  f2:
    lang: node
    handler: handler
    image: image
    limits:
      memory: 10m
      vendor.domain/fpga: 1
  f3:
    lang: node
    handler: handler
    image: image
    limits:
      cpu: 0.1
      memory: 10m
      vendor.domain/gpu: 1
      vendor.domain/fpga: 1
  f4:
    lang: node
    handler: handler
    image: image
    limits:
      vendor_1.domain/gpu: 1
      vendor_2.domain/gpu: 1
      vendor_3.domain/fpga: 1
  f5:
    lang: node
    handler: handler
    image: image
    limits:
      something/gpu: 1
  f6:
    lang: node
    handler: handler
    image: image
    limits:
      something/fpga: 1
  f7:
    lang: node
    handler: handler
    image: image
    limits:
      some.vendor/gpu: 1
      some.vendor/fastgpu: 1
  f8:
    lang: node
    handler: handler
    image: image
    limits:
      some.vendor/fpga: 1
      some.vendor/fastfpga: 1
  f9:
    lang: node
    handler: handler
    image: image
    limits:
      random: 1
`

const noMatchesErrorMsg string = "no functions matching --filter/--regex were found in the YAML file"
const invalidRegexErrorMsg string = "error parsing regexp"

var ParseYAMLTests_Regex = []struct {
	title         string
	searchTerm    string
	functions     []string
	file          string
	expectedError string
}{
	{
		title:         "Regex search for functions only containing 'node'",
		searchTerm:    "node",
		functions:     []string{"nodejs-echo"},
		file:          TestData_1,
		expectedError: "",
	},
	{
		title:         "Regex search for functions only containing 'echo'",
		searchTerm:    "echo",
		functions:     []string{"nodejs-echo", "ruby-echo"},
		file:          TestData_1,
		expectedError: "",
	},
	{
		title:         "Regex search for functions only containing '.+-.+'",
		searchTerm:    ".+-.+",
		functions:     []string{"abcd-eeee", "nodejs-echo", "ruby-echo", "url-ping"},
		file:          TestData_1,
		expectedError: "",
	},
	{
		title:         "Regex search for all functions: '.*'",
		searchTerm:    ".*",
		functions:     []string{"abcd-eeee", "imagemagick", "nodejs-echo", "ruby-echo", "url-ping"},
		file:          TestData_1,
		expectedError: "",
	},
	{
		title:         "Regex search for no functions: '----'",
		searchTerm:    "----",
		functions:     []string{},
		file:          TestData_1,
		expectedError: noMatchesErrorMsg,
	},
	{
		title:         "Regex search for functions without dashes: '^[^-]+$'",
		searchTerm:    "^[^-]+$",
		functions:     []string{"imagemagick"},
		file:          TestData_1,
		expectedError: "",
	},
	{
		title:         "Regex search for functions with 8 characters: '^.{8}$'",
		searchTerm:    "^.{8}$",
		functions:     []string{"url-ping"},
		file:          TestData_1,
		expectedError: "",
	},
	{
		title:         "Regex search for function with repeated 'e': 'e{2}'",
		searchTerm:    "e{2}",
		functions:     []string{"abcd-eeee"},
		file:          TestData_1,
		expectedError: "",
	},
	{
		title:         "Regex empty search term: ''",
		searchTerm:    "",
		functions:     []string{"abcd-eeee", "imagemagick", "nodejs-echo", "ruby-echo", "url-ping"},
		file:          TestData_1,
		expectedError: "",
	},
	{
		title:         "Regex invalid regex 1: '['",
		searchTerm:    "[",
		functions:     []string{},
		file:          TestData_1,
		expectedError: invalidRegexErrorMsg,
	},
	{
		title:         "Regex invalid regex 2: '*'",
		searchTerm:    "*",
		functions:     []string{},
		file:          TestData_1,
		expectedError: invalidRegexErrorMsg,
	},
	{
		title:         "Regex invalid regex 3: '(\\w)\\1'",
		searchTerm:    `(\w)\1`,
		functions:     []string{},
		file:          TestData_1,
		expectedError: invalidRegexErrorMsg,
	},
	{
		title:         "Regex that finds no matches: 'RANDOMREGEX'",
		searchTerm:    "RANDOMREGEX",
		functions:     []string{},
		file:          TestData_1,
		expectedError: noMatchesErrorMsg,
	},
	{
		title:         "Regex empty search term in empty YAML file: ",
		searchTerm:    "",
		functions:     []string{},
		file:          TestData_2,
		expectedError: "",
	},
}

var ParseYAMLTests_Filter = []struct {
	title         string
	searchTerm    string
	functions     []string
	file          string
	expectedError string
}{
	{
		title:         "Wildcard search for functions ending with 'echo'",
		searchTerm:    "*echo",
		functions:     []string{"nodejs-echo", "ruby-echo"},
		file:          TestData_1,
		expectedError: "",
	},
	{
		title:         "Wildcard search for functions with a - in between two words: '*-*'",
		searchTerm:    "*-*",
		functions:     []string{"abcd-eeee", "nodejs-echo", "ruby-echo", "url-ping"},
		file:          TestData_1,
		expectedError: "",
	},
	{
		title:         "Wildcard search for specific function: 'imagemagick'",
		searchTerm:    "imagemagick",
		functions:     []string{"imagemagick"},
		file:          TestData_1,
		expectedError: "",
	},
	{
		title:         "Wildcard search for all functions: '*'",
		searchTerm:    "*",
		functions:     []string{"abcd-eeee", "imagemagick", "nodejs-echo", "ruby-echo", "url-ping"},
		file:          TestData_1,
		expectedError: "",
	},
	{
		title:         "Wildcard empty search term: ''",
		searchTerm:    "",
		functions:     []string{"abcd-eeee", "imagemagick", "nodejs-echo", "ruby-echo", "url-ping"},
		file:          TestData_1,
		expectedError: "",
	},
	{
		title:         "Wildcard multiple wildcard characters: '**'",
		searchTerm:    "**",
		functions:     []string{"abcd-eeee", "imagemagick", "nodejs-echo", "ruby-echo", "url-ping"},
		file:          TestData_1,
		expectedError: "",
	},
	{
		title:         "Wildcard that finds no matches: 'RANDOMTEXT'",
		searchTerm:    "RANDOMTEXT",
		functions:     []string{},
		file:          TestData_1,
		expectedError: noMatchesErrorMsg,
	},
	{
		title:         "Wildcard empty search term in empty YAML file: ''",
		searchTerm:    "",
		functions:     []string{},
		file:          TestData_2,
		expectedError: "",
	},
}

func Test_ParseYAMLDataRegex(t *testing.T) {

	for _, test := range ParseYAMLTests_Regex {
		t.Run(test.title, func(t *testing.T) {

			parsedYAML, err := ParseYAMLData([]byte(test.file), test.searchTerm, "")

			if len(test.expectedError) > 0 {
				if err == nil {
					t.Errorf("Test_ParseYAMLDataRegex test [%s] test failed, expected error not thrown", test.title)
				}

				if !strings.Contains(err.Error(), test.expectedError) {
					t.Errorf("Test_ParseYAMLDataRegex test [%s] test failed, expected error message of '%s', got '%v'", test.title, test.expectedError, err)
				}

			} else {

				if err != nil {
					t.Errorf("Test_ParseYAMLDataRegex test [%s] test failed, unexpected error thrown: %v", test.title, err)
					return
				}

				keys := reflect.ValueOf(parsedYAML.Functions).MapKeys()
				strkeys := make([]string, len(keys))

				for i := 0; i < len(keys); i++ {
					strkeys[i] = keys[i].String()
				}

				sort.Strings(strkeys)
				t.Log(strkeys)

				if !reflect.DeepEqual(strkeys, test.functions) {
					t.Errorf("Test_ParseYAMLDataRegex test [%s] test failed, does not match expected result;\n  parsedYAML:   [%v]\n  expected: [%v]",
						test.title,
						strkeys,
						test.functions,
					)
				}
			}
		})
	}
}

func Test_ParseYAMLDataFilter(t *testing.T) {

	for _, test := range ParseYAMLTests_Filter {
		t.Run(test.title, func(t *testing.T) {

			parsedYAML, err := ParseYAMLData([]byte(test.file), "", test.searchTerm)

			if len(test.expectedError) > 0 {

				if err == nil {
					t.Errorf("Test_ParseYAMLDataFilter test [%s] test failed, expected error not thrown", test.title)
				}

				if !strings.Contains(err.Error(), test.expectedError) {
					t.Errorf("Test_ParseYAMLDataFilter test [%s] test failed, expected error message of '%s', got '%v'", test.title, test.expectedError, err)
				}

			} else {

				if err != nil {
					t.Errorf("Test_ParseYAMLDataFilter test [%s] test failed, unexpected error thrown: %v", test.title, err)
					return
				}

				keys := reflect.ValueOf(parsedYAML.Functions).MapKeys()
				strkeys := make([]string, len(keys))

				for i := 0; i < len(keys); i++ {
					strkeys[i] = keys[i].String()
				}

				sort.Strings(strkeys)
				t.Log(strkeys)

				if !reflect.DeepEqual(strkeys, test.functions) {
					t.Errorf("Test_ParseYAMLDataFilter test [%s] failed, does not match expected result;\n  parsedYAML:   [%v]\n  expected: [%v]",
						test.title,
						strkeys,
						test.functions,
					)
				}
			}
		})
	}
}

func Test_ParseYAMLDataFilterAndRegex(t *testing.T) {
	_, err := ParseYAMLData([]byte(TestData_1), ".*", "*")
	if err == nil {
		t.Errorf("Test_ParseYAMLDataFilterAndRegex test failed, expected error not thrown")
	}
}

func Test_ParseYAMLData_ProviderValues(t *testing.T) {
	testCases := []struct {
		title         string
		provider      string
		expectedError string
		file          string
	}{
		{
			title:         "Provider is faas and gives no error",
			provider:      "faas",
			expectedError: "",
			file: `provider:
  name: faas
  gateway: http://127.0.0.1:8080
  network: "func_functions"
`,
		},
		{
			title:         "Provider is openfaas and gives no error",
			provider:      "faas",
			expectedError: "",
			file: `provider:
  name: faas
  gateway: http://127.0.0.1:8080
  network: "func_functions"
`,
		},
		{
			title:         "Provider is serverless-openfaas and gives error",
			provider:      "faas",
			expectedError: "['faas', 'openfaas'] is the only valid provider for this tool - found: serverless-openfaas",
			file: `provider:
  name: serverless-openfaas
  gateway: http://127.0.0.1:8080
  network: "func_functions"
`,
		},
	}

	for _, test := range testCases {
		t.Run(test.title, func(t *testing.T) {

			_, err := ParseYAMLData([]byte(test.file), ".*", "*")
			if len(test.expectedError) > 0 {
				if test.expectedError != err.Error() {
					t.Errorf("want error: '%s', got: '%s'", test.expectedError, err.Error())
					t.Fail()
				}
			}
		})
	}
}

var ParseYAMLTests_ExtResources = []struct {
	title    string
	function string
	expected []string
}{
	{
		title:    "Valid resource: gpu",
		function: "f1",
		expected: []string{"vendor.domain/gpu"},
	},
	{
		title:    "Valid resource: fpga",
		function: "f2",
		expected: []string{"vendor.domain/fpga"},
	},
	{
		title:    "Valid resources: gpu, fpga",
		function: "f3",
		expected: []string{"vendor.domain/fpga", "vendor.domain/gpu"},
	},
	{
		title:    "Valid resources: gpu, gpu, fpga",
		function: "f4",
		expected: []string{"vendor_1.domain/gpu", "vendor_2.domain/gpu", "vendor_3.domain/fpga"},
	},
	{
		title:    "Resource specified with invalid domain: something/gpu",
		function: "f5",
		expected: []string{},
	},
	{
		title:    "Resource specified with invalid domain: something/fpga",
		function: "f6",
		expected: []string{},
	},
	{
		title:    "Invalid resource: fastgpu",
		function: "f7",
		expected: []string{"some.vendor/gpu"},
	},
	{
		title:    "Invalid resource: fastfpga",
		function: "f8",
		expected: []string{"some.vendor/fpga"},
	},
	{
		title:    "Invalid resource: random",
		function: "f9",
		expected: []string{},
	},
}

/**
 * Test parsing of extended resources, such as GPUs and FPGAs
 */
func Test_ParseYAMLData_ExtResources(t *testing.T) {

	for _, test := range ParseYAMLTests_ExtResources {
		t.Run(test.title, func(t *testing.T) {
			parsedYAML, err := ParseYAMLData([]byte(TestData_ExtResources), test.function, "")

			if err != nil {
				t.Errorf("Test_ParseYAMLData_ExtResources [%s] test failed: %v", test.title, err)
				return
			}

			for _, function := range parsedYAML.Functions {
				keys := reflect.ValueOf(function.Limits.Others).MapKeys()

				parsed := make([]string, len(keys))

				for i := 0; i < len(keys); i++ {
					parsed[i] = keys[i].String()
				}
				sort.Strings(parsed)

				if !reflect.DeepEqual(parsed, test.expected) {
					t.Errorf("Test_ParseYAMLData_ExtResources [%s] test failed, does not match expected result;\n  parsed resources:   [%v]\n  expected resources: [%v]",
						test.title,
						parsed,
						test.expected,
					)
				}

				t.Log(parsed)
			}
		})
	}
}
