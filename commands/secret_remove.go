// Copyright (c) OpenFaaS Author(s) 2019. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package commands

import (
	"context"
	"fmt"
	"github.com/openfaas/faas-cli/proxy"
	"github.com/openfaas/faas-provider/types"
	"github.com/spf13/cobra"
	"os"
)

var secretRemoveCmd = &cobra.Command{
	Use:     "remove [--tls-no-verify]",
	Aliases: []string{"rm"},
	Short:   "remove a secret",
	Long:    `Remove a secret by name`,
	Example: `faas-cli secret remove NAME
faas-cli secret remove NAME --gateway=http://127.0.0.1:8080`,
	RunE:    runSecretRemove,
	PreRunE: preRunSecretRemoveCmd,
}

func init() {
	secretRemoveCmd.Flags().StringVarP(&gateway, "gateway", "g", defaultGateway, "Gateway URL starting with http(s)://")
	secretRemoveCmd.Flags().BoolVar(&tlsInsecure, "tls-no-verify", false, "Disable TLS validation")
	secretRemoveCmd.Flags().StringVarP(&token, "token", "k", "", "Pass a JWT token to use instead of basic auth")
	secretRemoveCmd.Flags().StringVarP(&functionNamespace, "namespace", "n", "", "Namespace of the function")
	secretCmd.AddCommand(secretRemoveCmd)
}

func preRunSecretRemoveCmd(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("secret name required")
	}

	if len(args) > 1 {
		return fmt.Errorf("too many values for secret name")
	}
	return nil
}

func runSecretRemove(_ *cobra.Command, args []string) error {
	var gatewayAddress string
	gatewayAddress = getGatewayURL(gateway, defaultGateway, "", os.Getenv(openFaaSURLEnvironment))

	if msg := checkTLSInsecure(gatewayAddress, tlsInsecure); len(msg) > 0 {
		fmt.Println(msg)
	}

	secret := types.Secret{
		Name:      args[0],
		Namespace: functionNamespace,
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

	_, err = client.RemoveSecret(context.Background(), secret)

	if err != nil {
		return err
	}

	fmt.Println("Removed OK")

	return nil
}
