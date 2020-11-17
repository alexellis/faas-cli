package builder

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openfaas/faas-cli/stack"
)

func Test_isLanguageTemplate_Dockerfile(t *testing.T) {

	language := "Dockerfile"

	want := false
	got := isLanguageTemplate(language)
	if got != want {
		t.Errorf("language: %s got %v, want %v", language, got, want)
	}
}

func Test_isLanguageTemplate_Node(t *testing.T) {

	language := "node"

	want := true
	got := isLanguageTemplate(language)
	if got != want {
		t.Errorf("language: %s got %v, want %v", language, got, want)
	}
}

func Test_getDockerBuildCommand_NoOpts(t *testing.T) {
	dockerBuildVal := dockerBuild{
		Image:            "imagename:latest",
		NoCache:          false,
		Squash:           false,
		HTTPProxy:        "",
		HTTPSProxy:       "",
		BuildArgMap:      make(map[string]string),
		BuildOptPackages: []string{},
		BuildKitSecret:   []string{},
	}

	want := "build -t imagename:latest ."
	wantCommand := "docker"

	command, args := getDockerBuildCommand(dockerBuildVal)

	joined := strings.Join(args, " ")

	if joined != want {
		t.Errorf("getDockerBuildCommand want: \"%s\", got: \"%s\"", want, joined)
	}

	if command != wantCommand {
		t.Errorf("getDockerBuildCommand want command: \"%s\", got: \"%s\"", wantCommand, command)
	}
}

func Test_getDockerBuildCommand_WithNoCache(t *testing.T) {
	dockerBuildVal := dockerBuild{
		Image:            "imagename:latest",
		NoCache:          true,
		Squash:           false,
		HTTPProxy:        "",
		HTTPSProxy:       "",
		BuildArgMap:      make(map[string]string),
		BuildOptPackages: []string{},
		BuildKitSecret:   []string{},
	}

	want := "build --no-cache -t imagename:latest ."

	wantCommand := "docker"

	command, args := getDockerBuildCommand(dockerBuildVal)

	joined := strings.Join(args, " ")

	if joined != want {
		t.Errorf("getDockerBuildCommand want: \"%s\", got: \"%s\"", want, joined)
	}

	if command != wantCommand {
		t.Errorf("getDockerBuildCommand want command: \"%s\", got: \"%s\"", wantCommand, command)
	}
}

func Test_getDockerBuildCommand_WithProxies(t *testing.T) {
	dockerBuildVal := dockerBuild{
		Image:            "imagename:latest",
		NoCache:          false,
		Squash:           false,
		HTTPProxy:        "http://127.0.0.1:3128",
		HTTPSProxy:       "https://127.0.0.1:3128",
		BuildArgMap:      make(map[string]string),
		BuildOptPackages: []string{},
		BuildKitSecret:   []string{},
	}

	want := "build --build-arg http_proxy=http://127.0.0.1:3128 --build-arg https_proxy=https://127.0.0.1:3128 -t imagename:latest ."

	wantCommand := "docker"

	command, args := getDockerBuildCommand(dockerBuildVal)

	joined := strings.Join(args, " ")

	if joined != want {
		t.Errorf("getDockerBuildCommand want: \"%s\", got: \"%s\"", want, joined)
	}

	if command != wantCommand {
		t.Errorf("getDockerBuildCommand want command: \"%s\", got: \"%s\"", wantCommand, command)
	}
}

func Test_getDockerBuildCommand_WithBuildArg(t *testing.T) {
	dockerBuildVal := dockerBuild{
		Image:   "imagename:latest",
		NoCache: false,
		Squash:  false,
		BuildArgMap: map[string]string{
			"USERNAME": "admin",
			"PASSWORD": "1234",
		},
		BuildOptPackages: []string{},
		BuildKitSecret:   []string{},
	}

	_, values := getDockerBuildCommand(dockerBuildVal)

	joined := strings.Join(values, " ")
	wantArg1 := "--build-arg USERNAME=admin"
	wantArg2 := "--build-arg PASSWORD=1234"

	if strings.Contains(joined, wantArg1) == false {
		t.Errorf("want %s in %s, but didn't find it", wantArg1, joined)
	}
	if strings.Contains(joined, wantArg2) == false {
		t.Errorf("want %s in %s, but didn't find it", wantArg2, joined)
	}
}

func Test_getDockerBuildCommand_WithBuildKitSecret(t *testing.T) {
	dockerBuildVal := dockerBuild{
		Image:            "imagename:latest",
		NoCache:          false,
		Squash:           false,
		HTTPProxy:        "",
		HTTPSProxy:       "",
		BuildArgMap:      make(map[string]string),
		BuildOptPackages: []string{},
		BuildKitSecret:   []string{"id=s1,src=s1.txt"},
	}

	want := "build --secret id=s1,src=s1.txt -t imagename:latest ."

	wantCommand := "docker"

	command, args := getDockerBuildCommand(dockerBuildVal)

	joined := strings.Join(args, " ")

	if joined != want {
		t.Errorf("getDockerBuildCommand want: \"%s\", got: \"%s\"", want, joined)
	}

	if command != wantCommand {
		t.Errorf("getDockerBuildCommand want command: \"%s\", got: \"%s\"", wantCommand, command)
	}
}

func Test_buildFlagSlice(t *testing.T) {

	var buildFlagOpts = []struct {
		title          string
		nocache        bool
		squash         bool
		httpProxy      string
		httpsProxy     string
		buildArgMap    map[string]string
		buildPackages  []string
		buildkitSecret []string
		expectedSlice  []string
		buildLabelMap  map[string]string
	}{
		{
			title:          "no cache only",
			nocache:        true,
			squash:         false,
			httpProxy:      "",
			httpsProxy:     "",
			buildArgMap:    make(map[string]string),
			buildPackages:  []string{},
			buildkitSecret: []string{},
			expectedSlice:  []string{"--no-cache"},
		},
		{
			title:          "no cache & squash only",
			nocache:        true,
			squash:         true,
			httpProxy:      "",
			httpsProxy:     "",
			buildArgMap:    make(map[string]string),
			buildPackages:  []string{},
			buildkitSecret: []string{},
			expectedSlice:  []string{"--no-cache", "--squash"},
		},
		{
			title:          "no cache & squash & http proxy only",
			nocache:        true,
			squash:         true,
			httpProxy:      "192.168.0.1",
			httpsProxy:     "",
			buildArgMap:    make(map[string]string),
			buildPackages:  []string{},
			buildkitSecret: []string{},
			expectedSlice:  []string{"--no-cache", "--squash", "--build-arg", "http_proxy=192.168.0.1"},
		},
		{
			title:          "no cache & squash & https-proxy only",
			nocache:        true,
			squash:         true,
			httpProxy:      "",
			httpsProxy:     "127.0.0.1",
			buildArgMap:    make(map[string]string),
			buildPackages:  []string{},
			buildkitSecret: []string{},
			expectedSlice:  []string{"--no-cache", "--squash", "--build-arg", "https_proxy=127.0.0.1"},
		},
		{
			title:          "no cache & squash & http-proxy & https-proxy only",
			nocache:        true,
			squash:         true,
			httpProxy:      "192.168.0.1",
			httpsProxy:     "127.0.0.1",
			buildArgMap:    make(map[string]string),
			buildPackages:  []string{},
			buildkitSecret: []string{},
			expectedSlice:  []string{"--no-cache", "--squash", "--build-arg", "http_proxy=192.168.0.1", "--build-arg", "https_proxy=127.0.0.1"},
		},
		{
			title:          "http-proxy & https-proxy only",
			nocache:        false,
			squash:         false,
			httpProxy:      "192.168.0.1",
			httpsProxy:     "127.0.0.1",
			buildArgMap:    make(map[string]string),
			buildPackages:  []string{},
			buildkitSecret: []string{},
			expectedSlice:  []string{"--build-arg", "http_proxy=192.168.0.1", "--build-arg", "https_proxy=127.0.0.1"},
		},
		{
			title:      "build arg map no spaces",
			nocache:    false,
			squash:     false,
			httpProxy:  "",
			httpsProxy: "",
			buildArgMap: map[string]string{
				"muppet": "ernie",
			},
			buildPackages: []string{},
			expectedSlice: []string{"--build-arg", "muppet=ernie"},
		},
		{
			title:      "build arg map with spaces",
			nocache:    false,
			squash:     false,
			httpProxy:  "",
			httpsProxy: "",
			buildArgMap: map[string]string{
				"muppets": "burt and ernie",
			},
			buildPackages:  []string{},
			buildkitSecret: []string{},
			expectedSlice:  []string{"--build-arg", "muppets=burt and ernie"},
		},
		{
			title:      "multiple build arg map with spaces",
			nocache:    false,
			squash:     false,
			httpProxy:  "",
			httpsProxy: "",
			buildArgMap: map[string]string{
				"muppets":    "burt and ernie",
				"playschool": "Jemima",
			},
			buildPackages:  []string{},
			buildkitSecret: []string{},
			expectedSlice:  []string{"--build-arg", "muppets=burt and ernie", "--build-arg", "playschool=Jemima"},
		},
		{
			title:          "single secrets options map with docker buildkit build command secret option",
			nocache:        false,
			squash:         false,
			httpProxy:      "",
			httpsProxy:     "",
			buildArgMap:    make(map[string]string),
			buildPackages:  []string{},
			buildkitSecret: []string{"id=s1", "id=s2"},
			expectedSlice:  []string{"--secret", "id=s1", "--secret", "id=s2"},
		},
		{
			title:          "multi secrets options map with docker buildkit build command secret option",
			nocache:        false,
			squash:         false,
			httpProxy:      "",
			httpsProxy:     "",
			buildArgMap:    make(map[string]string),
			buildPackages:  []string{},
			buildkitSecret: []string{"id=s1,src=s1.txt", "id=s2,src=s2.txt"},
			expectedSlice:  []string{"--secret", "id=s1,src=s1.txt", "--secret", "id=s2,src=s2.txt"},
		},
		{
			title:      "no-cache and squash with multiple build arg map with spaces",
			nocache:    true,
			squash:     true,
			httpProxy:  "",
			httpsProxy: "",
			buildArgMap: map[string]string{
				"muppets":    "burt and ernie",
				"playschool": "Jemima",
			},
			buildPackages:  []string{},
			buildkitSecret: []string{},
			expectedSlice:  []string{"--no-cache", "--squash", "--build-arg", "muppets=burt and ernie", "--build-arg", "playschool=Jemima"},
		},
		{
			title:      "single build-label value",
			nocache:    false,
			squash:     false,
			httpProxy:  "",
			httpsProxy: "",
			buildArgMap: map[string]string{
				"muppets":    "burt and ernie",
				"playschool": "Jemima",
			},
			buildPackages:  []string{},
			buildkitSecret: []string{},
			buildLabelMap: map[string]string{
				"org.label-schema.name": "test function",
			},
			expectedSlice: []string{"--build-arg", "muppets=burt and ernie", "--build-arg", "playschool=Jemima", "--label", "org.label-schema.name=test function"},
		},
		{
			title:      "multiple build-label values",
			nocache:    false,
			squash:     false,
			httpProxy:  "",
			httpsProxy: "",
			buildArgMap: map[string]string{
				"muppets":    "burt and ernie",
				"playschool": "Jemima",
			},
			buildPackages:  []string{},
			buildkitSecret: []string{},
			buildLabelMap: map[string]string{
				"org.label-schema.name":        "test function",
				"org.label-schema.description": "This is a test function",
			},
			expectedSlice: []string{"--build-arg", "muppets=burt and ernie", "--build-arg", "playschool=Jemima", "--label", "org.label-schema.name=test function", "--label", "org.label-schema.description=This is a test function"},
		},
	}

	for _, test := range buildFlagOpts {

		t.Run(test.title, func(t *testing.T) {

			flagSlice := buildFlagSlice(test.nocache, test.squash, test.httpProxy, test.httpsProxy, test.buildArgMap, test.buildPackages, test.buildLabelMap, test.buildkitSecret)
			fmt.Println(flagSlice)
			if len(flagSlice) != len(test.expectedSlice) {
				t.Errorf("Slices differ in size - wanted: %d, found %d", len(test.expectedSlice), len(flagSlice))
			}

			isMatch := compareSliceValues(test.expectedSlice, flagSlice)
			if !isMatch {
				t.Errorf("Slices differ in values - wanted: %v, found %s", test.expectedSlice, flagSlice)
			}

		})
	}

}

func compareSliceValues(expectedSlice, actualSlice []string) bool {
	var expectedValueMap = make(map[string]int)
	for _, expectedValue := range expectedSlice {
		if _, ok := expectedValueMap[expectedValue]; ok {
			expectedValueMap[expectedValue]++
		} else {
			expectedValueMap[expectedValue] = 1
		}
	}

	var actualValueMap = make(map[string]int)
	for _, actualValue := range actualSlice {
		if _, ok := actualValueMap[actualValue]; ok {
			actualValueMap[actualValue]++
		} else {
			actualValueMap[actualValue] = 1
		}
	}

	if len(expectedValueMap) != len(actualValueMap) {
		return false
	}

	for key, expectedValue := range expectedValueMap {
		actualValue, exists := actualValueMap[key]
		if !exists || actualValue != expectedValue {
			return false
		}
	}

	return true
}

func Test_getPackages(t *testing.T) {
	var buildOpts = []struct {
		title                 string
		availableBuildOptions []stack.BuildOption
		requestedBuildOptions []string
		expectedPackages      []string
	}{
		{
			title: "Single Option",
			availableBuildOptions: []stack.BuildOption{
				stack.BuildOption{Name: "dev",
					Packages: []string{"jq", "hw", "ke"}},
			},
			requestedBuildOptions: []string{"dev"},
			expectedPackages:      []string{"jq", "hw", "ke"},
		},
		{
			title: "Two Options one chosen",
			availableBuildOptions: []stack.BuildOption{
				stack.BuildOption{Name: "dev",
					Packages: []string{"jq", "hw", "ke"}},
				stack.BuildOption{Name: "debug",
					Packages: []string{"lr", "kt", "jy"}},
			},
			requestedBuildOptions: []string{"dev"},
			expectedPackages:      []string{"jq", "hw", "ke"},
		},
		{
			title: "Two Options two chosen",
			availableBuildOptions: []stack.BuildOption{
				stack.BuildOption{Name: "dev",
					Packages: []string{"jq", "hw", "ke"}},
				stack.BuildOption{Name: "debug",
					Packages: []string{"lr", "kt", "jy"}},
			},
			requestedBuildOptions: []string{"dev", "debug"},
			expectedPackages:      []string{"jq", "hw", "ke", "lr", "kt", "jy"},
		},
		{
			title: "Two Options two chosen with overlaps",
			availableBuildOptions: []stack.BuildOption{
				stack.BuildOption{Name: "dev",
					Packages: []string{"jq", "hw", "ke"}},
				stack.BuildOption{Name: "debug",
					Packages: []string{"lr", "jq", "hw"}},
			},
			requestedBuildOptions: []string{"dev", "debug"},
			expectedPackages:      []string{"jq", "hw", "ke", "lr"},
		},
	}
	for _, test := range buildOpts {

		t.Run(test.title, func(t *testing.T) {

			buildOptPackages, _ := getPackages(test.availableBuildOptions, test.requestedBuildOptions)

			if len(buildOptPackages) != len(test.expectedPackages) {
				t.Errorf("Slices differ in size - wanted: %d, found %d", len(test.expectedPackages), len(buildOptPackages))
			}
			for _, expectedOptPackage := range test.expectedPackages {
				found := false
				for _, buildOptPackage := range buildOptPackages {
					if buildOptPackage == expectedOptPackage {
						found = true
						break
					}
				}
				if found == false {

					t.Errorf("Slices differ in values  - wanted: %s, found %s", strings.Join(test.expectedPackages, " "), strings.Join(buildOptPackages, " "))
				}
			}
		})
	}
}

func Test_deDuplicate(t *testing.T) {
	var stringOpts = []struct {
		title           string
		inputStrings    []string
		expectedStrings []string
	}{
		{
			title:           "No Duplicates",
			inputStrings:    []string{"jq", "hw", "ke"},
			expectedStrings: []string{"jq", "hw", "ke"},
		},
		{
			title:           "Duplicates",
			inputStrings:    []string{"jq", "hw", "ke", "jq", "hw", "ke"},
			expectedStrings: []string{"jq", "hw", "ke"},
		},
	}
	for _, test := range stringOpts {

		t.Run(test.title, func(t *testing.T) {

			uniqueStrings := deDuplicate(test.inputStrings)

			if len(uniqueStrings) != len(test.expectedStrings) {
				t.Errorf("Slices differ in size - wanted: %d, found %d", len(test.expectedStrings), len(uniqueStrings))
			}

			for _, expectedString := range test.expectedStrings {

				found := false

				for _, uniqueString := range uniqueStrings {

					if expectedString == uniqueString {
						found = true
						break
					}
				}

				if found == false {
					t.Errorf("Slices differ in values  - wanted: %s, found %s", strings.Join(test.expectedStrings, " "), strings.Join(uniqueStrings, " "))
				}
			}
		})
	}
}

func Test_pathInScope(t *testing.T) {
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("unexpected error during test setup: %s", err)
	}

	cases := []struct {
		name         string
		path         string
		expectedPath string
		err          bool
	}{
		{
			name:         "can copy folders without any relative path prefix",
			path:         "common/models/prebuilt",
			expectedPath: filepath.Join(root, "common/models/prebuilt"),
		},
		{
			name:         "can copy folders with relative path prefix",
			path:         "./common/data/cleaned",
			expectedPath: filepath.Join(root, "./common/data/cleaned"),
		},
		{
			name: "error if path equals the current directory",
			path: "./",
			err:  true,
		},
		{
			name: "error if relative path moves out of the current directory",
			path: "../private",
			err:  true,
		},
		{
			name: "error if absolute path is outside of the current directory",
			path: "/private/common",
			err:  true,
		},
		{
			name: "error if relative path moves out of the current directory, even when hidden in the middle of the path",
			path: "./common/../../private",
			err:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			abs, err := pathInScope(tc.path, root)
			switch {
			case tc.err && err == nil:
				t.Fatalf("expected error but got none")
			case tc.err && !strings.HasPrefix(err.Error(), "forbidden path"):
				t.Fatalf("expected forbidden path error, got \"%s\"", err)
			case !tc.err && err != nil:
				t.Fatalf("unexpected err \"%s\"", err)
			default:
				if abs != tc.expectedPath {
					t.Fatalf("expected path %s, got %s", tc.expectedPath, abs)
				}
			}
		})
	}
}
