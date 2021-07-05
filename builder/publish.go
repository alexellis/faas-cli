// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package builder

import (
	"fmt"
	"os"
	"strings"

	v1execute "github.com/alexellis/go-execute/pkg/v1"
	"github.com/openfaas/faas-cli/schema"
	"github.com/openfaas/faas-cli/stack"
)

// PublishImage will publish images as multi-arch
// TODO: refactor signature to a struct to simplify the length of the method header
func PublishImage(image string, handler string, functionName string, language string, nocache bool, squash bool, shrinkwrap bool, buildArgMap map[string]string,
	buildOptions []string, tagMode schema.BuildFormat, buildLabelMap map[string]string, quietBuild bool, copyExtraPaths []string, ignorePaths []string, platforms string, extraTags []string) error {

	if stack.IsValidTemplate(language) {
		pathToTemplateYAML := fmt.Sprintf("./template/%s/template.yml", language)
		if _, err := os.Stat(pathToTemplateYAML); os.IsNotExist(err) {
			return err
		}

		langTemplate, err := stack.ParseYAMLForLanguageTemplate(pathToTemplateYAML)
		if err != nil {
			return fmt.Errorf("error reading language template: %s", err.Error())
		}

		branch, version, err := GetImageTagValues(tagMode)
		if err != nil {
			return err
		}

		imageName := schema.BuildImageName(tagMode, image, version, branch)

		if err := ensureHandlerPath(handler); err != nil {
			return fmt.Errorf("building %s, %s is an invalid path", imageName, handler)
		}

		tempPath, buildErr := createBuildContext(functionName, handler, language, isLanguageTemplate(language), langTemplate.HandlerFolder, copyExtraPaths, ignorePaths)
		fmt.Printf("Building: %s with %s template. Please wait..\n", imageName, language)
		if buildErr != nil {
			return buildErr
		}

		if shrinkwrap {
			fmt.Printf("%s shrink-wrapped to %s\n", functionName, tempPath)
			return nil
		}

		buildOptPackages, buildPackageErr := getBuildOptionPackages(buildOptions, language, langTemplate.BuildOptions)

		if buildPackageErr != nil {
			return buildPackageErr

		}

		dockerBuildVal := dockerBuild{
			Image:            imageName,
			NoCache:          nocache,
			Squash:           squash,
			HTTPProxy:        os.Getenv("http_proxy"),
			HTTPSProxy:       os.Getenv("https_proxy"),
			BuildArgMap:      buildArgMap,
			BuildOptPackages: buildOptPackages,
			BuildLabelMap:    buildLabelMap,
			Platforms:        platforms,
			ExtraTags:        extraTags,
		}

		command, args := getDockerBuildxCommand(dockerBuildVal)
		fmt.Printf("Publishing with command: %v %v\n", command, args)

		task := v1execute.ExecTask{
			Cwd:         tempPath,
			Command:     command,
			Args:        args,
			StreamStdio: !quietBuild,
		}

		res, err := task.Execute()

		if err != nil {
			return err
		}

		if res.ExitCode != 0 {
			return fmt.Errorf("[%s] received non-zero exit code from build, error: %s", functionName, res.Stderr)
		}

		fmt.Printf("Image: %s built.\n", imageName)

	} else {
		return fmt.Errorf("language template: %s not supported, build a custom Dockerfile", language)
	}

	return nil
}

func getDockerBuildxCommand(build dockerBuild) (string, []string) {
	flagSlice := buildFlagSlice(build.NoCache, build.Squash, build.HTTPProxy, build.HTTPSProxy, build.BuildArgMap,
		build.BuildOptPackages, build.BuildLabelMap)

	// pushOnly defined at https://github.com/docker/buildx
	const pushOnly = "--output=type=registry,push=true"

	args := []string{"buildx", "build", "--progress=plain", "--platform=" + build.Platforms, pushOnly}

	args = append(args, flagSlice...)

	args = append(args, "--tag", build.Image, ".")

	for _, t := range build.ExtraTags {

		var tag string
		if i := strings.LastIndex(build.Image, ":"); i > -1 {
			tag = applyTag(i, build.Image, t)
		} else {
			tag = applyTag(len(build.Image)-1, build.Image, t)
		}
		args = append(args, "--tag", tag)
	}

	command := "docker"

	return command, args
}

func applyTag(index int, baseImage, tag string) string {
	return fmt.Sprintf("%s:%s", baseImage[:index], tag)
}
