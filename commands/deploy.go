// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package commands

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/openfaas/faas-cli/builder"
	"github.com/openfaas/faas-cli/proxy"
	"github.com/openfaas/faas-cli/schema"
	"github.com/openfaas/faas-cli/stack"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

var (
	// readTemplate controls whether we should read the function's template when deploying.
	readTemplate bool
)

// DeployFlags holds flags that are to be added to commands.
type DeployFlags struct {
	envvarOpts             []string
	replace                bool
	update                 bool
	readOnlyRootFilesystem bool
	constraints            []string
	secrets                []string
	labelOpts              []string
	annotationOpts         []string
}

var deployFlags DeployFlags

func init() {
	// Setup flags that are used by multiple commands (variables defined in faas.go)
	deployCmd.Flags().StringVar(&fprocess, "fprocess", "", "fprocess value to be run as a serverless function by the watchdog")
	deployCmd.Flags().StringVarP(&gateway, "gateway", "g", defaultGateway, "Gateway URL starting with http(s)://")
	deployCmd.Flags().StringVar(&handler, "handler", "", "Directory with handler for function, e.g. handler.js")
	deployCmd.Flags().StringVar(&image, "image", "", "Docker image name to build")
	deployCmd.Flags().StringVar(&language, "lang", "", "Programming language template")
	deployCmd.Flags().StringVar(&functionName, "name", "", "Name of the deployed function")
	deployCmd.Flags().StringVar(&network, "network", defaultNetwork, "Name of the network")
	deployCmd.Flags().StringVarP(&functionNamespace, "namespace", "n", "", "Namespace of the function")

	// Setup flags that are used only by this command (variables defined above)
	deployCmd.Flags().StringArrayVarP(&deployFlags.envvarOpts, "env", "e", []string{}, "Set one or more environment variables (ENVVAR=VALUE)")

	deployCmd.Flags().StringArrayVarP(&deployFlags.labelOpts, "label", "l", []string{}, "Set one or more label (LABEL=VALUE)")

	deployCmd.Flags().StringArrayVarP(&deployFlags.annotationOpts, "annotation", "", []string{}, "Set one or more annotation (ANNOTATION=VALUE)")

	deployCmd.Flags().BoolVar(&deployFlags.replace, "replace", false, "Remove and re-create existing function(s)")
	deployCmd.Flags().BoolVar(&deployFlags.update, "update", true, "Perform rolling update on existing function(s)")

	deployCmd.Flags().StringArrayVar(&deployFlags.constraints, "constraint", []string{}, "Apply a constraint to the function")
	deployCmd.Flags().StringArrayVar(&deployFlags.secrets, "secret", []string{}, "Give the function access to a secure secret")
	deployCmd.Flags().BoolVar(&deployFlags.readOnlyRootFilesystem, "readonly", false, "Force the root container filesystem to be read only")

	deployCmd.Flags().Var(&tagFormat, "tag", "Override latest tag on function Docker image, accepts 'latest', 'sha', 'branch', or 'describe'")

	deployCmd.Flags().BoolVar(&tlsInsecure, "tls-no-verify", false, "Disable TLS validation")
	deployCmd.Flags().BoolVar(&envsubst, "envsubst", true, "Substitute environment variables in stack.yml file")
	deployCmd.Flags().StringVarP(&token, "token", "k", "", "Pass a JWT token to use instead of basic auth")
	// Set bash-completion.
	_ = deployCmd.Flags().SetAnnotation("handler", cobra.BashCompSubdirsInDir, []string{})
	deployCmd.Flags().BoolVar(&readTemplate, "read-template", true, "Read the function's template")

	faasCmd.AddCommand(deployCmd)
}

// deployCmd handles deploying OpenFaaS function containers
var deployCmd = &cobra.Command{
	Use: `deploy -f YAML_FILE [--replace=false]
  faas-cli deploy --image IMAGE_NAME
                  --name FUNCTION_NAME
                  [--lang <ruby|python|node|csharp>]
                  [--gateway GATEWAY_URL]
                  [--network NETWORK_NAME]
                  [--handler HANDLER_DIR]
                  [--fprocess PROCESS]
                  [--env ENVVAR=VALUE ...]
				  [--label LABEL=VALUE ...]
				  [--annotation ANNOTATION=VALUE ...]
				  [--replace=false]
				  [--update=false]
                  [--constraint PLACEMENT_CONSTRAINT ...]
                  [--regex "REGEX"]
                  [--filter "WILDCARD"]
				  [--secret "SECRET_NAME"]
				  [--tag <sha|branch|describe>]
				  [--readonly=false]
				  [--tls-no-verify]`,

	Short: "Deploy OpenFaaS functions",
	Long: `Deploys OpenFaaS function containers either via the supplied YAML config using
the "--yaml" flag (which may contain multiple function definitions), or directly
via flags. Note: --replace and --update are mutually exclusive.`,
	Example: `  faas-cli deploy -f https://domain/path/myfunctions.yml
  faas-cli deploy -f ./stack.yml
  faas-cli deploy -f ./stack.yml --label canary=true
  faas-cli deploy -f ./stack.yml --annotation user=true
  faas-cli deploy -f ./stack.yml --filter "*gif*" --secret dockerhuborg
  faas-cli deploy -f ./stack.yml --regex "fn[0-9]_.*"
  faas-cli deploy -f ./stack.yml --replace=false --update=true
  faas-cli deploy -f ./stack.yml --replace=true --update=false
  faas-cli deploy -f ./stack.yml --tag sha
  faas-cli deploy -f ./stack.yml --tag branch
  faas-cli deploy -f ./stack.yml --tag describe
  faas-cli deploy --image=alexellis/faas-url-ping --name=url-ping
  faas-cli deploy --image=my_image --name=my_fn --handler=/path/to/fn/
                  --gateway=http://remote-site.com:8080 --lang=python
                  --env=MYVAR=myval`,
	PreRunE: preRunDeploy,
	RunE:    runDeploy,
}

// preRunDeploy validates args & flags
func preRunDeploy(cmd *cobra.Command, args []string) error {
	language, _ = validateLanguageFlag(language)

	return nil
}

func runDeploy(cmd *cobra.Command, args []string) error {
	return runDeployCommand(args, image, fprocess, functionName, deployFlags, tagFormat)
}

func runDeployCommand(args []string, image string, fprocess string, functionName string, deployFlags DeployFlags, tagMode schema.BuildFormat) error {
	if deployFlags.update && deployFlags.replace {
		fmt.Println(`Cannot specify --update and --replace at the same time. One of --update or --replace must be false.
  --replace    removes an existing deployment before re-creating it
  --update     performs a rolling update to a new function image or configuration (default true)`)
		return fmt.Errorf("cannot specify --update and --replace at the same time")
	}

	var services stack.Services
	if len(yamlFile) > 0 {
		parsedServices, err := stack.ParseYAMLFile(yamlFile, regex, filter, envsubst)
		if err != nil {
			return err
		}

		parsedServices.Provider.GatewayURL = getGatewayURL(gateway, defaultGateway, parsedServices.Provider.GatewayURL, os.Getenv(openFaaSURLEnvironment))

		if parsedServices != nil {
			services = *parsedServices
		}
	}

	transport := GetDefaultCLITransport(tlsInsecure, &commandTimeout)
	ctx := context.Background()

	var failedStatusCodes = make(map[string]int)
	if len(services.Functions) > 0 {

		cliAuth, err := proxy.NewCLIAuth(token, services.Provider.GatewayURL)
		if err != nil {
			return err
		}
		proxyClient, err := proxy.NewClient(cliAuth, services.Provider.GatewayURL, transport, &commandTimeout)
		if err != nil {
			return err
		}

		for k, function := range services.Functions {

			functionSecrets := deployFlags.secrets

			function.Name = k
			fmt.Printf("Deploying: %s.\n", function.Name)

			var functionConstraints []string
			if function.Constraints != nil {
				functionConstraints = *function.Constraints
			} else if len(deployFlags.constraints) > 0 {
				functionConstraints = deployFlags.constraints
			}

			if len(function.Secrets) > 0 {
				functionSecrets = mergeSlice(function.Secrets, functionSecrets)
			}

			// Check if there is a functionNamespace flag passed, if so, override the namespace value
			// defined in the stack.yaml
			function.Namespace = getNamespace(functionNamespace, function.Namespace)

			fileEnvironment, err := readFiles(function.EnvironmentFile)
			if err != nil {
				return err
			}

			labelMap := map[string]string{}
			if function.Labels != nil {
				labelMap = *function.Labels
			}

			labelArgumentMap, labelErr := parseMap(deployFlags.labelOpts, "label")
			if labelErr != nil {
				return fmt.Errorf("error parsing labels: %v", labelErr)
			}

			allLabels := mergeMap(labelMap, labelArgumentMap)

			allEnvironment, envErr := compileEnvironment(deployFlags.envvarOpts, function.Environment, fileEnvironment)
			if envErr != nil {
				return envErr
			}

			if readTemplate {
				// Get FProcess to use from the ./template/template.yml, if a template is being used
				if languageExistsNotDockerfile(function.Language) {
					var fprocessErr error

					function.FProcess, fprocessErr = deriveFprocess(function)
					if fprocessErr != nil {
						return fmt.Errorf(`template directory may be missing or invalid, please run "faas-cli template pull"
Error: %s`, fprocessErr.Error())
					}
				}
			}

			functionResourceRequest := proxy.FunctionResourceRequest{
				Limits:   function.Limits,
				Requests: function.Requests,
			}

			var annotations map[string]string
			if function.Annotations != nil {
				annotations = *function.Annotations
			}

			annotationArgs, annotationErr := parseMap(deployFlags.annotationOpts, "annotation")
			if annotationErr != nil {
				return fmt.Errorf("error parsing annotations: %v", annotationErr)
			}

			allAnnotations := mergeMap(annotations, annotationArgs)

			branch, sha, err := builder.GetImageTagValues(tagMode)
			if err != nil {
				return err
			}

			function.Image = schema.BuildImageName(tagMode, function.Image, sha, branch)

			if deployFlags.readOnlyRootFilesystem {
				function.ReadOnlyRootFilesystem = deployFlags.readOnlyRootFilesystem
			}

			deploySpec := &proxy.DeployFunctionSpec{
				FProcess:                function.FProcess,
				FunctionName:            function.Name,
				Image:                   function.Image,
				Language:                function.Language,
				Replace:                 deployFlags.replace,
				EnvVars:                 allEnvironment,
				Constraints:             functionConstraints,
				Update:                  deployFlags.update,
				Secrets:                 functionSecrets,
				Labels:                  allLabels,
				Annotations:             allAnnotations,
				FunctionResourceRequest: functionResourceRequest,
				ReadOnlyRootFilesystem:  function.ReadOnlyRootFilesystem,
				TLSInsecure:             tlsInsecure,
				Token:                   token,
				Namespace:               function.Namespace,
			}

			if msg := checkTLSInsecure(services.Provider.GatewayURL, deploySpec.TLSInsecure); len(msg) > 0 {
				fmt.Println(msg)
			}
			dRes, res, err := proxyClient.DeployFunction(ctx, deploySpec)

			if err == nil {
				fmt.Println(dRes.Message)
				fmt.Printf("URL: %s\n\n", dRes.URL)
			}

			if badStatusCode(res.StatusCode) {
				failedStatusCodes[k] = res.StatusCode
			}
		}
	} else {
		if len(image) == 0 || len(functionName) == 0 {
			return fmt.Errorf("To deploy a function give --yaml/-f or a --image and --name flag")
		}
		gateway = getGatewayURL(gateway, defaultGateway, "", os.Getenv(openFaaSURLEnvironment))
		cliAuth, err := proxy.NewCLIAuth(token, gateway)
		if err != nil {
			return err
		}
		proxyClient, err := proxy.NewClient(cliAuth, gateway, transport, &commandTimeout)
		if err != nil {
			return err
		}

		// default to a readable filesystem until we get more input about the expected behavior
		// and if we want to add another flag for this case
		defaultReadOnlyRFS := false
		statusCode, err := deployImage(ctx, proxyClient, image, fprocess, functionName, "", deployFlags,
			tlsInsecure, defaultReadOnlyRFS, token, functionNamespace)
		if err != nil {
			return err
		}

		if badStatusCode(statusCode) {
			failedStatusCodes[functionName] = statusCode
		}
	}

	if err := deployFailed(failedStatusCodes); err != nil {
		return err
	}

	return nil
}

// deployImage deploys a function with the given image
func deployImage(
	ctx context.Context,
	client *proxy.Client,
	image string,
	fprocess string,
	functionName string,
	registryAuth string,
	deployFlags DeployFlags,
	tlsInsecure bool,
	readOnlyRootFilesystem bool,
	token string,
	namespace string,
) (int, error) {

	var statusCode int
	readOnlyRFS := deployFlags.readOnlyRootFilesystem || readOnlyRootFilesystem
	envvars, err := parseMap(deployFlags.envvarOpts, "env")

	if err != nil {
		return statusCode, fmt.Errorf("error parsing envvars: %v", err)
	}

	labelMap, labelErr := parseMap(deployFlags.labelOpts, "label")

	if labelErr != nil {
		return statusCode, fmt.Errorf("error parsing labels: %v", labelErr)
	}

	annotationMap, annotationErr := parseMap(deployFlags.annotationOpts, "annotation")

	if annotationErr != nil {
		return statusCode, fmt.Errorf("error parsing annotations: %v", annotationErr)
	}

	deploySpec := &proxy.DeployFunctionSpec{
		FProcess:                fprocess,
		FunctionName:            functionName,
		Image:                   image,
		Language:                language,
		Replace:                 deployFlags.replace,
		EnvVars:                 envvars,
		Network:                 network,
		Constraints:             deployFlags.constraints,
		Update:                  deployFlags.update,
		Secrets:                 deployFlags.secrets,
		Labels:                  labelMap,
		Annotations:             annotationMap,
		FunctionResourceRequest: proxy.FunctionResourceRequest{},
		ReadOnlyRootFilesystem:  readOnlyRFS,
		TLSInsecure:             tlsInsecure,
		Token:                   token,
		Namespace:               namespace,
	}

	if msg := checkTLSInsecure(gateway, deploySpec.TLSInsecure); len(msg) > 0 {
		fmt.Println(msg)
	}

	dRes, res, err := client.DeployFunction(ctx, deploySpec)
	if err != nil {
		return res.StatusCode, err
	}

	fmt.Println(dRes.Message)
	fmt.Printf("URL: %s\n\n", dRes.URL)
	return res.StatusCode, nil
}

func mergeSlice(values []string, overlay []string) []string {
	results := []string{}
	added := make(map[string]bool)
	for _, value := range overlay {
		results = append(results, value)
		added[value] = true
	}

	for _, value := range values {
		if exists := added[value]; exists == false {
			results = append(results, value)
		}
	}

	return results
}

func readFiles(files []string) (map[string]string, error) {
	envs := make(map[string]string)

	for _, file := range files {
		bytesOut, readErr := ioutil.ReadFile(file)
		if readErr != nil {
			return nil, readErr
		}

		envFile := stack.EnvironmentFile{}
		unmarshalErr := yaml.Unmarshal(bytesOut, &envFile)
		if unmarshalErr != nil {
			return nil, unmarshalErr
		}
		for k, v := range envFile.Environment {
			envs[k] = v
		}
	}
	return envs, nil
}

func parseMap(envvars []string, keyName string) (map[string]string, error) {
	result := make(map[string]string)
	for _, envvar := range envvars {
		s := strings.SplitN(strings.TrimSpace(envvar), "=", 2)
		if len(s) != 2 {
			return nil, fmt.Errorf("label format is not correct, needs key=value")
		}
		envvarName := s[0]
		envvarValue := s[1]

		if !(len(envvarName) > 0) {
			return nil, fmt.Errorf("empty %s name: [%s]", keyName, envvar)
		}
		if !(len(envvarValue) > 0) {
			return nil, fmt.Errorf("empty %s value: [%s]", keyName, envvar)
		}

		result[envvarName] = envvarValue
	}
	return result, nil
}

func mergeMap(i map[string]string, j map[string]string) map[string]string {
	merged := make(map[string]string)

	for k, v := range i {
		merged[k] = v
	}
	for k, v := range j {
		merged[k] = v
	}
	return merged
}

func compileEnvironment(envvarOpts []string, yamlEnvironment map[string]string, fileEnvironment map[string]string) (map[string]string, error) {
	envvarArguments, err := parseMap(envvarOpts, "env")
	if err != nil {
		return nil, fmt.Errorf("error parsing envvars: %v", err)
	}

	functionAndStack := mergeMap(yamlEnvironment, fileEnvironment)
	return mergeMap(functionAndStack, envvarArguments), nil
}

func deriveFprocess(function stack.Function) (string, error) {
	var fprocess string

	pathToTemplateYAML := "./template/" + function.Language + "/template.yml"
	if _, err := os.Stat(pathToTemplateYAML); os.IsNotExist(err) {
		return "", err
	}

	var langTemplate stack.LanguageTemplate
	parsedLangTemplate, err := stack.ParseYAMLForLanguageTemplate(pathToTemplateYAML)

	if err != nil {
		return "", err

	}

	if parsedLangTemplate != nil {
		langTemplate = *parsedLangTemplate
		fprocess = langTemplate.FProcess
	}

	return fprocess, nil
}

func languageExistsNotDockerfile(language string) bool {
	return len(language) > 0 && strings.ToLower(language) != "dockerfile"
}

type authConfig struct {
	Auth string `json:"auth,omitempty"`
}

type configFile struct {
	AuthConfigs      map[string]authConfig `json:"auths"`
	CredentialsStore string                `json:"credsStore,omitempty"`
}

const (
	// docker default settings
	configFileName        = "config.json"
	configFileDir         = ".docker"
	defaultDockerRegistry = "https://index.docker.io/v1/"
)

var (
	configDir = os.Getenv("DOCKER_CONFIG")
)

func deployFailed(status map[string]int) error {
	if len(status) == 0 {
		return nil
	}

	var allErrors []string
	for funcName, funcStatus := range status {
		err := fmt.Errorf("Function '%s' failed to deploy with status code: %d", funcName, funcStatus)
		allErrors = append(allErrors, err.Error())
	}
	return fmt.Errorf(strings.Join(allErrors, "\n"))
}

func badStatusCode(statusCode int) bool {
	return statusCode != http.StatusAccepted && statusCode != http.StatusOK
}
