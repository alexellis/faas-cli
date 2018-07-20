// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package builder

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/morikuni/aec"
)

// ExecCommand run a system command
func ExecCommand(tempPath string, builder []string) {
	targetCmd := exec.Command(builder[0], builder[1:]...)
	targetCmd.Dir = tempPath
	targetCmd.Stdout = os.Stdout
	targetCmd.Stderr = os.Stderr
	targetCmd.Start()
	err := targetCmd.Wait()
	if err != nil {
		errString := fmt.Sprintf("ERROR - Could not execute command: %s", builder)
		log.Fatalf(aec.RedF.Apply(errString))
	}
}

// ExecCommand run a system command an return stdout
func ExecCommandWithOutput(builder []string, skipFailure bool) string {
	output, err := exec.Command(builder[0], builder[1:]...).CombinedOutput()
	if err != nil && !skipFailure {
		errString := fmt.Sprintf("ERROR - Could not execute command: %s", builder)
		log.Fatalf(aec.RedF.Apply(errString))
	}
	return string(output)
}

//Generate image version of type gittag-gitsha
func GetVersion() string {
	getShaCommand := []string{"git", "rev-parse", "--short", "HEAD"}
	sha := ExecCommandWithOutput(getShaCommand, true)
	if strings.Contains(sha, "Not a git repository") {
		return ""
	}
	sha = strings.TrimSuffix(sha, "\n")

	getTagCommand := []string{"git", "tag", "--points-at", sha}
	tag := ExecCommandWithOutput(getTagCommand, true)
	tag = strings.TrimSuffix(tag, "\n")
	if len(tag) == 0 {
		tag = "latest"
	}

	return ":" + tag + "-" + sha
}
