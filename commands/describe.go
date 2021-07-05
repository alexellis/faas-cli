// Copyright (c) OpenFaaS Author(s) 2018. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package commands

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/openfaas/faas-cli/proxy"
	"github.com/openfaas/faas-cli/schema"
	"github.com/openfaas/faas-cli/stack"

	"github.com/spf13/cobra"
)

func init() {
	describeCmd.Flags().StringVar(&functionName, "name", "", "Name of the function")
	describeCmd.Flags().StringVarP(&gateway, "gateway", "g", defaultGateway, "Gateway URL starting with http(s)://")
	describeCmd.Flags().BoolVar(&tlsInsecure, "tls-no-verify", false, "Disable TLS validation")
	describeCmd.Flags().BoolVar(&envsubst, "envsubst", true, "Substitute environment variables in stack.yml file")
	describeCmd.Flags().StringVarP(&token, "token", "k", "", "Pass a JWT token to use instead of basic auth")
	describeCmd.Flags().StringVarP(&functionNamespace, "namespace", "n", "", "Namespace of the function")

	faasCmd.AddCommand(describeCmd)
}

var describeCmd = &cobra.Command{
	Use:   "describe FUNCTION_NAME [--gateway GATEWAY_URL]",
	Short: "Describe an OpenFaaS function",
	Long:  `Display details of an OpenFaaS function`,
	Example: `faas-cli describe figlet 
faas-cli describe env --gateway http://127.0.0.1:8080
faas-cli describe echo -g http://127.0.0.1.8080`,
	PreRunE: preRunDescribe,
	RunE:    runDescribe,
}

func preRunDescribe(cmd *cobra.Command, args []string) error {
	return nil
}

func runDescribe(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("please provide a name for the function")
	}
	var yamlGateway string
	var services stack.Services
	functionName = args[0]

	if len(yamlFile) > 0 {
		parsedServices, err := stack.ParseYAMLFile(yamlFile, regex, filter, envsubst)
		if err != nil {
			return err
		}

		if parsedServices != nil {
			services = *parsedServices
			yamlGateway = services.Provider.GatewayURL
		}
	}
	gatewayAddress := getGatewayURL(gateway, defaultGateway, yamlGateway, os.Getenv(openFaaSURLEnvironment))
	cliAuth, err := proxy.NewCLIAuth(token, gatewayAddress)
	if err != nil {
		return err
	}
	transport := GetDefaultCLITransport(tlsInsecure, &commandTimeout)
	cliClient, err := proxy.NewClient(cliAuth, gatewayAddress, transport, &commandTimeout)
	if err != nil {
		return err
	}

	ctx := context.Background()

	function, err := cliClient.GetFunctionInfo(ctx, functionName, functionNamespace)
	if err != nil {
		return err
	}

	//To get correct value for invocation count from /system/functions endpoint
	functionList, _, err := cliClient.ListFunctions(ctx, functionNamespace)
	if err != nil {
		return actionableErrorMessage(err)
	}

	var invocationCount int
	for _, fn := range functionList {
		if fn.Name == function.Name {
			invocationCount = int(fn.InvocationCount)
			break
		}
	}

	var status = "Not Ready"
	if function.AvailableReplicas > 0 {
		status = "Ready"
	}

	url, asyncURL := getFunctionURLs(gatewayAddress, functionName, functionNamespace)

	funcDesc := schema.FunctionDescription{
		Name:              function.Name,
		Status:            status,
		Replicas:          int(function.Replicas),
		AvailableReplicas: int(function.AvailableReplicas),
		InvocationCount:   int(invocationCount),
		Image:             function.Image,
		EnvProcess:        function.EnvProcess,
		URL:               url,
		AsyncURL:          asyncURL,
		Labels:            function.Labels,
		Annotations:       function.Annotations,
	}

	printFunctionDescription(funcDesc)

	return nil
}

func getFunctionURLs(gateway string, functionName string, functionNamespace string) (string, string) {
	gateway = strings.TrimRight(gateway, "/")

	url := gateway + "/function/" + functionName
	asyncURL := gateway + "/async-function/" + functionName

	if functionNamespace != "" {
		url += "." + functionNamespace
		asyncURL += "." + functionNamespace
	}

	return url, asyncURL
}

func printFunctionDescription(funcDesc schema.FunctionDescription) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.TabIndent)
	fmt.Fprintln(w, "Name:\t "+funcDesc.Name)
	fmt.Fprintln(w, "Status:\t "+funcDesc.Status)
	fmt.Fprintln(w, "Replicas:\t "+strconv.Itoa(funcDesc.Replicas))
	fmt.Fprintln(w, "Available replicas:\t "+strconv.Itoa(funcDesc.AvailableReplicas))
	fmt.Fprintln(w, "Invocations:\t "+strconv.Itoa(funcDesc.InvocationCount))
	fmt.Fprintln(w, "Image:\t "+funcDesc.Image)
	fmt.Fprintln(w, "Function process:\t "+funcDesc.EnvProcess)
	fmt.Fprintln(w, "URL:\t "+funcDesc.URL)
	fmt.Fprintln(w, "Async URL:\t "+funcDesc.AsyncURL)

	if funcDesc.Labels != nil {
		fmt.Fprintf(w, "Labels:")
		for key, value := range *funcDesc.Labels {
			fmt.Fprintln(w, " \t "+key+" : "+value)
		}
	}

	if funcDesc.Annotations != nil {
		fmt.Fprintf(w, "Annotations:")
		for key, value := range *funcDesc.Annotations {
			fmt.Fprintln(w, " \t "+key+" : "+value)
		}
	}
	w.Flush()
}
