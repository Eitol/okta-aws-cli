/*
 * Copyright (c) 2023-Present, Okta, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package web

import (
	"github.com/spf13/cobra"

	"github.com/okta/okta-aws-cli/internal/config"
	cliFlag "github.com/okta/okta-aws-cli/internal/flag"
	"github.com/okta/okta-aws-cli/internal/okta"
	"github.com/okta/okta-aws-cli/internal/webssoauth"
)

var (
	flags = []cliFlag.Flag{
		{
			Name:   config.AWSAcctFedAppIDFlag,
			Short:  "a",
			Value:  "",
			Usage:  "AWS Account Federation app ID",
			EnvVar: config.OktaAWSAccountFederationAppIDEnvVar,
		},
		{
			Name:   config.AWSIAMIdPFlag,
			Short:  "i",
			Value:  "",
			Usage:  "Preset IAM Identity Provider ARN",
			EnvVar: config.AWSIAMIdPEnvVar,
		},
		{
			Name:   config.QRCodeFlag,
			Short:  "q",
			Value:  false,
			Usage:  "Print QR Code of activation URL",
			EnvVar: config.QRCodeEnvVar,
		},
		{
			Name:   config.OpenBrowserFlag,
			Short:  "b",
			Value:  false,
			Usage:  "Automatically open the activation URL with the system web browser",
			EnvVar: config.OpenBrowserEnvVar,
		},
		{
			Name:   config.OpenBrowserCommandFlag,
			Short:  "m",
			Value:  "",
			Usage:  "Automatically open the activation URL with the given web browser command",
			EnvVar: config.OpenBrowserCommandEnvVar,
		},
		{
			Name:   config.AllProfilesFlag,
			Short:  "k",
			Value:  false,
			Usage:  "Collect all profiles for a given IdP (implies aws-credentials file output format)",
			EnvVar: config.AllProfilesEnvVar,
		},
	}
	requiredFlags = []interface{}{"org-domain", "oidc-client-id"}
)

// NewWebCommand Sets up the web cobra sub command
func NewWebCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "web",
		Short: "Human oriented authentication and device authorization",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.EvaluateSettings()
			if err != nil {
				return err
			}

			// Warn if there is an issue with okta.yaml
			_, err = config.OktaConfig()
			if err != nil {
				webssoauth.ConsolePrint(cfg, "WARNING: issue with %s file. Run `okta-aws-cli debug` command for additional diagnosis.\nError: %+v\n", config.OktaYaml, err)
			}

			err = cliFlag.CheckRequiredFlags(requiredFlags)
			if err != nil {
				return err
			}

			for attempt := 1; attempt <= 2; attempt++ {
				wsa, err := webssoauth.NewWebSSOAuthentication(cfg)
				if _, ok := err.(*webssoauth.ClassicOrgError); ok {
					return err
				}
				if err != nil {
					break
				}

				err = wsa.EstablishIAMCredentials()
				if err == nil {
					break
				}

				if apiErr, ok := err.(*okta.APIError); ok {
					if apiErr.ErrorType == "invalid_grant" && webssoauth.RemoveCachedAccessToken() {
						webssoauth.ConsolePrint(cfg, "\nCached access token appears to be stale, removing token and retrying device authorization ...\n\n")
						continue
					}
					break
				}
			}

			return err
		},
	}

	cliFlag.MakeFlagBindings(cmd, flags, false)

	return cmd
}
