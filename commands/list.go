// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package commands

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/openfaas/faas-cli/proxy"
	"github.com/openfaas/faas-cli/stack"
	"github.com/openfaas/faas-provider/types"
	"github.com/spf13/cobra"
)

var (
	verboseList bool
	token       string
	sortOrder   string
)

func init() {
	// Setup flags that are used by multiple commands (variables defined in faas.go)
	listCmd.Flags().StringVarP(&gateway, "gateway", "g", defaultGateway, "Gateway URL starting with http(s)://")
	listCmd.Flags().StringVarP(&functionNamespace, "namespace", "n", "", "Namespace of the function")
	listCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Quiet mode - print out only the function's ID")

	listCmd.Flags().BoolVarP(&verboseList, "verbose", "v", false, "Verbose output for the function list")
	listCmd.Flags().BoolVar(&tlsInsecure, "tls-no-verify", false, "Disable TLS validation")
	listCmd.Flags().BoolVar(&envsubst, "envsubst", true, "Substitute environment variables in stack.yml file")
	listCmd.Flags().StringVarP(&token, "token", "k", "", "Pass a JWT token to use instead of basic auth")
	listCmd.Flags().StringVar(&sortOrder, "sort", "name", "Sort the functions by \"name\" or \"invocations\"")

	faasCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:     `list [--gateway GATEWAY_URL] [--verbose] [--tls-no-verify]`,
	Aliases: []string{"ls"},
	Short:   "List OpenFaaS functions",
	Long:    `Lists OpenFaaS functions either on a local or remote gateway`,
	Example: `  faas-cli list
  faas-cli list --gateway https://127.0.0.1:8080 --verbose`,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	var services stack.Services
	var gatewayAddress string
	var yamlGateway string
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
	gatewayAddress = getGatewayURL(gateway, defaultGateway, yamlGateway, os.Getenv(openFaaSURLEnvironment))

	cliAuth, err := proxy.NewCLIAuth(token, gatewayAddress)
	if err != nil {
		return err
	}
	transport := GetDefaultCLITransport(tlsInsecure, &commandTimeout)
	proxyClient, err := proxy.NewClient(cliAuth, gatewayAddress, transport, &commandTimeout)
	if err != nil {
		return err
	}

	functions, _, err := proxyClient.ListFunctions(context.Background(), functionNamespace)
	if err != nil {
		return actionableErrorMessage(err)
	}

	if sortOrder == "name" {
		sort.Sort(byName(functions))
	} else if sortOrder == "invocations" {
		sort.Sort(byInvocations(functions))
	} else if sortOrder == "creation" {
		sort.Sort(byCreation(functions))
	}

	if quiet {
		for _, function := range functions {
			fmt.Printf("%s\n", function.Name)
		}
	} else if verboseList {

		maxWidth := 40
		for _, function := range functions {
			if len(function.Image) > maxWidth {
				maxWidth = len(function.Image)
			}
		}

		fmt.Printf("%-30s\t%-"+fmt.Sprintf("%d", maxWidth)+"s\t%-15s\t%-5s\t%-5s\n", "Function", "Image", "Invocations", "Replicas", "CreatedAt")
		for _, function := range functions {
			functionImage := function.Image
			// if len(function.Image) > 40 {
			// 	functionImage = functionImage[0:38] + ".."
			// }
			fmt.Printf("%-30s\t%-"+fmt.Sprintf("%d", maxWidth)+"s\t%-15d\t%-5d\t\t%-5s\n", function.Name, functionImage, int64(function.InvocationCount), function.Replicas, function.CreatedAt.String())
		}
	} else {
		fmt.Printf("%-30s\t%-15s\t%-5s\n", "Function", "Invocations", "Replicas")
		for _, function := range functions {
			fmt.Printf("%-30s\t%-15d\t%-5d\n", function.Name, int64(function.InvocationCount), function.Replicas)
		}
	}
	return nil
}

type byName []types.FunctionStatus

func (a byName) Len() int           { return len(a) }
func (a byName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byName) Less(i, j int) bool { return a[i].Name < a[j].Name }

type byInvocations []types.FunctionStatus

func (a byInvocations) Len() int           { return len(a) }
func (a byInvocations) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byInvocations) Less(i, j int) bool { return a[i].InvocationCount > a[j].InvocationCount }

type byCreation []types.FunctionStatus

func (a byCreation) Len() int           { return len(a) }
func (a byCreation) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byCreation) Less(i, j int) bool { return a[i].CreatedAt.Before(a[j].CreatedAt) }
