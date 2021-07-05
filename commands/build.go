// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package commands

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/morikuni/aec"
	"github.com/openfaas/faas-cli/builder"
	"github.com/openfaas/faas-cli/schema"
	"github.com/openfaas/faas-cli/stack"
	"github.com/openfaas/faas-cli/versioncontrol"
	"github.com/spf13/cobra"
)

// Flags that are to be added to commands.
var (
	nocache          bool
	squash           bool
	parallel         int
	shrinkwrap       bool
	buildArgs        []string
	buildArgMap      map[string]string
	buildOptions     []string
	copyExtra        []string
	tagFormat        schema.BuildFormat
	buildLabels      []string
	buildLabelMap    map[string]string
	envsubst         bool
	quietBuild       bool
	disableStackPull bool
)

func init() {
	// Setup flags that are used by multiple commands (variables defined in faas.go)
	buildCmd.Flags().StringVar(&image, "image", "", "Docker image name to build")
	buildCmd.Flags().StringVar(&handler, "handler", "", "Directory with handler for function, e.g. handler.js")
	buildCmd.Flags().StringVar(&functionName, "name", "", "Name of the deployed function")
	buildCmd.Flags().StringVar(&language, "lang", "", "Programming language template")

	// Setup flags that are used only by this command (variables defined above)
	buildCmd.Flags().BoolVar(&nocache, "no-cache", false, "Do not use Docker's build cache")
	buildCmd.Flags().BoolVar(&squash, "squash", false, `Use Docker's squash flag for smaller images [experimental] `)
	buildCmd.Flags().IntVar(&parallel, "parallel", 1, "Build in parallel to depth specified.")
	buildCmd.Flags().BoolVar(&shrinkwrap, "shrinkwrap", false, "Just write files to ./build/ folder for shrink-wrapping")
	buildCmd.Flags().StringArrayVarP(&buildArgs, "build-arg", "b", []string{}, "Add a build-arg for Docker (KEY=VALUE)")
	buildCmd.Flags().StringArrayVarP(&buildOptions, "build-option", "o", []string{}, "Set a build option, e.g. dev")
	buildCmd.Flags().Var(&tagFormat, "tag", "Override latest tag on function Docker image, accepts 'latest', 'sha', 'branch', or 'describe'")
	buildCmd.Flags().StringArrayVar(&buildLabels, "build-label", []string{}, "Add a label for Docker image (LABEL=VALUE)")
	buildCmd.Flags().StringArrayVar(&copyExtra, "copy-extra", []string{}, "Extra paths that will be copied into the function build context")
	buildCmd.Flags().BoolVar(&envsubst, "envsubst", true, "Substitute environment variables in stack.yml file")
	buildCmd.Flags().BoolVar(&quietBuild, "quiet", false, "Perform a quiet build, without showing output from Docker")
	buildCmd.Flags().BoolVar(&disableStackPull, "disable-stack-pull", false, "Disables the template configuration in the stack.yml")

	// Set bash-completion.
	_ = buildCmd.Flags().SetAnnotation("handler", cobra.BashCompSubdirsInDir, []string{})

	faasCmd.AddCommand(buildCmd)
}

// buildCmd allows the user to build an OpenFaaS function container
var buildCmd = &cobra.Command{
	Use: `build -f YAML_FILE [--no-cache] [--squash]
  faas-cli build --image IMAGE_NAME
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
                 [--tag <sha|branch|describe>]`,
	Short: "Builds OpenFaaS function containers",
	Long: `Builds OpenFaaS function containers either via the supplied YAML config using
the "--yaml" flag (which may contain multiple function definitions), or directly
via flags.`,
	Example: `  faas-cli build -f https://domain/path/myfunctions.yml
  faas-cli build -f ./stack.yml --no-cache --build-arg NPM_VERSION=0.2.2
  faas-cli build -f ./stack.yml --build-option dev
  faas-cli build -f ./stack.yml --tag sha
  faas-cli build -f ./stack.yml --tag branch
  faas-cli build -f ./stack.yml --tag describe
  faas-cli build -f ./stack.yml --filter "*gif*"
  faas-cli build -f ./stack.yml --regex "fn[0-9]_.*"
  faas-cli build --image=my_image --lang=python --handler=/path/to/fn/
                 --name=my_fn --squash
  faas-cli build -f ./stack.yml --build-label org.label-schema.label-name="value"`,
	PreRunE: preRunBuild,
	RunE:    runBuild,
}

// preRunBuild validates args & flags
func preRunBuild(cmd *cobra.Command, args []string) error {
	language, _ = validateLanguageFlag(language)

	mapped, err := parseBuildArgs(buildArgs)

	if err == nil {
		buildArgMap = mapped
	}

	buildLabelMap, err = parseMap(buildLabels, "build-label")

	if parallel < 1 {
		return fmt.Errorf("the --parallel flag must be great than 0")
	}

	return err
}

func parseBuildArgs(args []string) (map[string]string, error) {
	mapped := make(map[string]string)

	for _, kvp := range args {
		index := strings.Index(kvp, "=")
		if index == -1 {
			return nil, fmt.Errorf("each build-arg must take the form key=value")
		}

		values := []string{kvp[0:index], kvp[index+1:]}

		k := strings.TrimSpace(values[0])
		v := strings.TrimSpace(values[1])

		if len(k) == 0 {
			return nil, fmt.Errorf("build-arg must have a non-empty key")
		}
		if len(v) == 0 {
			return nil, fmt.Errorf("build-arg must have a non-empty value")
		}

		if k == builder.AdditionalPackageBuildArg && len(mapped[k]) > 0 {
			mapped[k] = mapped[k] + " " + v
		} else {
			mapped[k] = v
		}
	}

	return mapped, nil
}

func runBuild(cmd *cobra.Command, args []string) error {

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

	if len(services.Functions) == 0 {
		if len(image) == 0 {
			return fmt.Errorf("please provide a valid --image name for your Docker image")
		}
		if len(handler) == 0 {
			return fmt.Errorf("please provide the full path to your function's handler")
		}
		if len(functionName) == 0 {
			return fmt.Errorf("please provide the deployed --name of your function")
		}
		err := builder.BuildImage(image,
			handler,
			functionName,
			language,
			nocache,
			squash,
			shrinkwrap,
			buildArgMap,
			buildOptions,
			tagFormat,
			buildLabelMap,
			quietBuild,
			copyExtra,
			nil,
		)
		if err != nil {
			return err
		}
		return nil
	}

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

	errors := build(&services, parallel, shrinkwrap, quietBuild)
	if len(errors) > 0 {
		errorSummary := "Errors received during build:\n"
		for _, err := range errors {
			errorSummary = errorSummary + "- " + err.Error() + "\n"
		}
		return fmt.Errorf("%s", aec.Apply(errorSummary, aec.RedF))
	}
	return nil
}

func build(services *stack.Services, queueDepth int, shrinkwrap, quietBuild bool) []error {
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
					err := builder.BuildImage(function.Image,
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

// PullTemplates pulls templates from specified git remote. templateURL may be a pinned repository.
func PullTemplates(templateURL string) error {
	var err error
	exists, err := os.Stat("./template")
	if err != nil || exists == nil {
		log.Println("No templates found in current directory.")

		templateURL, refName := versioncontrol.ParsePinnedRemote(templateURL)
		err = fetchTemplates(templateURL, refName, false)
		if err != nil {
			log.Println("Unable to download templates from Github.")
			return err
		}
	}
	return err
}

func combineBuildOpts(YAMLBuildOpts []string, buildFlagBuildOpts []string) []string {

	return mergeSlice(YAMLBuildOpts, buildFlagBuildOpts)

}
