// Copyright (c) OpenFaaS Author(s) 2019. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package commands

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/openfaas/faas-cli/proxy"
	types "github.com/openfaas/faas-provider/types"
	"github.com/spf13/cobra"
)

var (
	literalSecret string
	secretFile    string
	trimSecret    bool
)

// secretCreateCmd represents the secretCreate command
var secretCreateCmd = &cobra.Command{
	Use: `create SECRET_NAME
			[--trim=false]
			[--from-literal=SECRET_VALUE]
			[--from-file=/path/to/secret/file]
			[STDIN]
			[--tls-no-verify]`,
	Short: "Create a new secret",
	Long:  `The create command creates a new secret from file, literal or STDIN`,
	Example: `faas-cli secret create secret-name --from-literal=secret-value
faas-cli secret create secret-name --from-literal=secret-value --gateway=http://127.0.0.1:8080
faas-cli secret create secret-name --from-file=/path/to/secret/file --gateway=http://127.0.0.1:8080
cat /path/to/secret/file | faas-cli secret create secret-name`,
	RunE:    runSecretCreate,
	PreRunE: preRunSecretCreate,
}

func init() {
	secretCreateCmd.Flags().StringVar(&literalSecret, "from-literal", "", "Value of the secret")
	secretCreateCmd.Flags().StringVar(&secretFile, "from-file", "", "Path to the secret file")
	secretCreateCmd.Flags().BoolVar(&trimSecret, "trim", true, "trim whitespace from the start and end of the secret value")
	secretCreateCmd.Flags().BoolVar(&tlsInsecure, "tls-no-verify", false, "Disable TLS validation")
	secretCreateCmd.Flags().StringVarP(&gateway, "gateway", "g", defaultGateway, "Gateway URL starting with http(s)://")
	secretCreateCmd.Flags().StringVarP(&token, "token", "k", "", "Pass a JWT token to use instead of basic auth")
	secretCreateCmd.Flags().StringVarP(&functionNamespace, "namespace", "n", "", "Namespace of the function")
	secretCmd.AddCommand(secretCreateCmd)
}

func preRunSecretCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("secret name required")
	}

	if len(args) > 1 {
		return fmt.Errorf("too many values for secret name")
	}

	if len(secretFile) > 0 && len(literalSecret) > 0 {
		return fmt.Errorf("please provide secret using only one option from --from-literal, --from-file and STDIN")
	}

	isValid, err := validateSecretName(args[0])
	if !isValid {
		return err
	}

	return nil
}

func runSecretCreate(cmd *cobra.Command, args []string) error {
	secret := types.Secret{
		Name:      args[0],
		Namespace: functionNamespace,
	}

	switch {
	case len(literalSecret) > 0:
		secret.Value = literalSecret

	case len(secretFile) > 0:
		var err error
		secret.Value, err = readSecretFromFile(secretFile)
		if err != nil {
			return err
		}

	default:
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			fmt.Fprintf(os.Stderr, "Reading from STDIN - hit (Control + D) to stop.\n")
		}

		secretStdin, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		secret.Value = string(secretStdin)
	}

	if trimSecret {
		secret.Value = strings.TrimSpace(secret.Value)
	}

	if len(secret.Value) == 0 {
		return fmt.Errorf("must provide a non empty secret via --from-literal, --from-file or STDIN")
	}

	gatewayAddress := getGatewayURL(gateway, defaultGateway, "", os.Getenv(openFaaSURLEnvironment))

	if msg := checkTLSInsecure(gatewayAddress, tlsInsecure); len(msg) > 0 {
		fmt.Println(msg)
	}
	cliAuth, err := proxy.NewCLIAuth(token, gatewayAddress)
	if err != nil {
		return err
	}
	transport := GetDefaultCLITransport(tlsInsecure, &commandTimeout)
	client, err := proxy.NewClient(cliAuth, gatewayAddress, transport, &commandTimeout)
	if err != nil {
		return err
	}

	fmt.Println("Creating secret: " + secret.Name)
	_, err = client.CreateSecret(context.Background(), secret)
	if err != nil {
		return err
	}
	fmt.Println("Secret Created")

	return nil
}

func readSecretFromFile(secretFile string) (string, error) {
	fileData, err := ioutil.ReadFile(secretFile)
	return string(fileData), err
}

// Kubernetes DNS-1123 Subdomain Regex
// https://github.com/kubernetes/kubernetes/blob/6902f3112d98eb6bd0894886ff9cd3fbd03a7f79/staging/src/k8s.io/apimachinery/pkg/util/validation/validation.go#L131
const (
	dns1123LabelFmt          string = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	dns1123SubdomainFmt      string = dns1123LabelFmt + "(\\." + dns1123LabelFmt + ")*"
	invalidSecretNameMessage string = "ERROR: invalid secret name %s\nSecret name must start and end with an alphanumeric character \nand can only contain lower-case alphanumeric characters, '-' or '.'"
)

func validateSecretName(secretName string) (bool, error) {
	var dns1123SubdomainRegexp = regexp.MustCompile("^" + dns1123SubdomainFmt + "$")

	if !dns1123SubdomainRegexp.MatchString(secretName) {
		return false, fmt.Errorf(invalidSecretNameMessage, secretName)
	}

	return true, nil
}
