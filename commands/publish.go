// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package commands

import (
	"fmt"
	"os"
	"sync"
	"time"

	v1execute "github.com/alexellis/go-execute/pkg/v1"
	"github.com/morikuni/aec"
	"github.com/openfaas/faas-cli/builder"
	"github.com/openfaas/faas-cli/stack"
	"github.com/spf13/cobra"
)

var (
	platforms string
	extraTags []string
)

func init() {
	// Setup flags that are used by multiple commands (variables defined in faas.go)
	publishCmd.Flags().StringVar(&image, "image", "", "Docker image name to build")
	publishCmd.Flags().StringVar(&handler, "handler", "", "Directory with handler for function, e.g. handler.js")
	publishCmd.Flags().StringVar(&functionName, "name", "", "Name of the deployed function")
	publishCmd.Flags().StringVar(&language, "lang", "", "Programming language template")

	// Setup flags that are used only by this command (variables defined above)
	publishCmd.Flags().BoolVar(&nocache, "no-cache", false, "Do not use Docker's build cache")
	publishCmd.Flags().BoolVar(&squash, "squash", false, `Use Docker's squash flag for smaller images [experimental] `)
	publishCmd.Flags().IntVar(&parallel, "parallel", 1, "Build in parallel to depth specified.")
	publishCmd.Flags().BoolVar(&shrinkwrap, "shrinkwrap", false, "Just write files to ./build/ folder for shrink-wrapping")
	publishCmd.Flags().StringArrayVarP(&buildArgs, "build-arg", "b", []string{}, "Add a build-arg for Docker (KEY=VALUE)")
	publishCmd.Flags().StringArrayVarP(&buildOptions, "build-option", "o", []string{}, "Set a build option, e.g. dev")
	publishCmd.Flags().Var(&tagFormat, "tag", "Override latest tag on function Docker image, accepts 'latest', 'sha', 'branch', or 'describe'")
	publishCmd.Flags().StringArrayVar(&buildLabels, "build-label", []string{}, "Add a label for Docker image (LABEL=VALUE)")
	publishCmd.Flags().StringArrayVar(&copyExtra, "copy-extra", []string{}, "Extra paths that will be copied into the function build context")
	publishCmd.Flags().BoolVar(&envsubst, "envsubst", true, "Substitute environment variables in stack.yml file")
	publishCmd.Flags().BoolVar(&quietBuild, "quiet", false, "Perform a quiet build, without showing output from Docker")
	publishCmd.Flags().BoolVar(&disableStackPull, "disable-stack-pull", false, "Disables the template configuration in the stack.yml")
	publishCmd.Flags().StringVar(&platforms, "platforms", "linux/amd64", "A set of platforms to publish")
	publishCmd.Flags().StringArrayVar(&extraTags, "extra-tag", []string{}, "Additional extra image tag")

	// Set bash-completion.
	_ = publishCmd.Flags().SetAnnotation("handler", cobra.BashCompSubdirsInDir, []string{})

	faasCmd.AddCommand(publishCmd)
}

// publishCmd allows the user to build an OpenFaaS function container
var publishCmd = &cobra.Command{
	Use: `publish -f YAML_FILE [--no-cache] [--squash]
  faas-cli publish --image IMAGE_NAME
                   --handler HANDLER_DIR
                   --name FUNCTION_NAME
                   [--lang <ruby|python|python3|node|csharp|dockerfile>]
                   [--no-cache] [--squash]
                   [--regex "REGEX"]
                   [--filter "WILDCARD"]
                   [--parallel PARALLEL_DEPTH]
                   [--build-arg KEY=VALUE]
                   [--build-option VALUE]
                   [--copy-extra PATH]
                   [--tag <sha|branch|describe>]
                   [--platforms linux/arm/v7]`,
	Short: "Builds and pushes multi-arch OpenFaaS container images",
	Long: `Builds and pushes multi-arch OpenFaaS container images using Docker buildx.
Most users will want faas-cli build or faas-cli up for development and testing.
This command is designed to make releasing and publishing multi-arch container
images easier.

A stack.yaml file is required, and any images that are built will not be
available in the local Docker library. This is due to technical constraints in
Docker and buildx. You must use a multi-arch template to use this command with
correctly configured TARGETPLATFORM and BUILDPLATFORM arguments.

See also: faas-cli build`,
	Example: `  faas-cli publish --platforms linux/amd64,linux/arm64,linux/arm/7
  faas-cli publish --platforms linux/arm/7 --filter webhook
  faas-cli publish -f go.yml --no-cache --build-arg NPM_VERSION=0.2.2
  faas-cli publish --build-option dev
  faas-cli publish --tag sha
  `,
	PreRunE: preRunPublish,
	RunE:    runPublish,
}

// preRunPublish validates args & flags
func preRunPublish(cmd *cobra.Command, args []string) error {
	language, _ = validateLanguageFlag(language)

	mapped, err := parseBuildArgs(buildArgs)

	if err == nil {
		buildArgMap = mapped
	}

	buildLabelMap, err = parseMap(buildLabels, "build-label")

	if parallel < 1 {
		return fmt.Errorf("the --parallel flag must be great than 0")
	}

	if len(yamlFile) == 0 {
		return fmt.Errorf("--yaml or -f is required")
	}

	return err
}

func runPublish(cmd *cobra.Command, args []string) error {

	var services stack.Services
	if len(yamlFile) > 0 {
		parsedServices, err := stack.ParseYAMLFile(yamlFile, regex, filter, envsubst)
		if err != nil {
			return err
		}

		if parsedServices != nil {
			services = *parsedServices
		}
	}

	templateAddress := getTemplateURL("", os.Getenv(templateURLEnvironment), DefaultTemplateRepository)
	if pullErr := PullTemplates(templateAddress); pullErr != nil {
		return fmt.Errorf("could not pull templates for OpenFaaS: %v", pullErr)
	}

	task := v1execute.ExecTask{
		Command:     "docker",
		Args:        []string{"buildx", "create", "--use", "--name=multiarch", "--node=multiarch"},
		StreamStdio: !quietBuild,
		Env:         []string{"DOCKER_CLI_EXPERIMENTAL=enabled"},
	}

	res, err := task.Execute()
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("non-zero exit code: %d", res.ExitCode)
	}

	fmt.Printf("Created buildx node: %s\n", res.Stdout)

	if len(services.StackConfiguration.TemplateConfigs) != 0 && !disableStackPull {
		newTemplateInfos, err := filterExistingTemplates(services.StackConfiguration.TemplateConfigs, "./template")
		if err != nil {
			return fmt.Errorf("Already pulled templates directory has issue: %s", err.Error())
		}

		err = pullStackTemplates(newTemplateInfos, cmd)
		if err != nil {
			return fmt.Errorf("could not pull templates from function yaml file: %s", err.Error())
		}
	}

	errors := publish(&services, parallel, shrinkwrap, quietBuild)
	if len(errors) > 0 {
		errorSummary := "Errors received during build:\n"
		for _, err := range errors {
			errorSummary = errorSummary + "- " + err.Error() + "\n"
		}
		return fmt.Errorf("%s", aec.Apply(errorSummary, aec.RedF))
	}
	return nil
}

func publish(services *stack.Services, queueDepth int, shrinkwrap, quietBuild bool) []error {
	startOuter := time.Now()

	errors := []error{}

	wg := sync.WaitGroup{}

	workChannel := make(chan stack.Function)

	wg.Add(queueDepth)
	for i := 0; i < queueDepth; i++ {
		go func(index int) {
			for function := range workChannel {
				start := time.Now()

				fmt.Printf(aec.YellowF.Apply("[%d] > Building %s.\n"), index, function.Name)
				if len(function.Language) == 0 {
					fmt.Println("Please provide a valid language for your function.")
				} else {
					combinedBuildOptions := combineBuildOpts(function.BuildOptions, buildOptions)
					combinedBuildArgMap := mergeMap(function.BuildArgs, buildArgMap)
					combinedExtraPaths := mergeSlice(services.StackConfiguration.CopyExtraPaths, copyExtra)
					err := builder.PublishImage(function.Image,
						function.Handler,
						function.Name,
						function.Language,
						nocache,
						squash,
						shrinkwrap,
						combinedBuildArgMap,
						combinedBuildOptions,
						tagFormat,
						buildLabelMap,
						quietBuild,
						combinedExtraPaths,
						function.IgnorePaths,
						platforms,
						extraTags,
					)

					if err != nil {
						errors = append(errors, err)
					}
				}

				duration := time.Since(start)
				fmt.Printf(aec.YellowF.Apply("[%d] < Building %s done in %1.2fs.\n"), index, function.Name, duration.Seconds())
			}

			fmt.Printf(aec.YellowF.Apply("[%d] Worker done.\n"), index)
			wg.Done()
		}(i)

	}

	for k, function := range services.Functions {
		if function.SkipBuild {
			fmt.Printf("Skipping build of: %s.\n", function.Name)
		} else {
			function.Name = k
			workChannel <- function
		}
	}

	close(workChannel)

	wg.Wait()

	duration := time.Since(startOuter)
	fmt.Printf("\n%s\n", aec.Apply(fmt.Sprintf("Total build time: %1.2fs", duration.Seconds()), aec.YellowF))
	return errors
}
