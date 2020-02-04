// Copyright (c) OpenFaaS Author(s) 2018. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package commands

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bep/debounce"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	skipPush   bool
	skipDeploy bool
	watch      bool
)

func init() {

	upFlagset := pflag.NewFlagSet("up", pflag.ExitOnError)
	upFlagset.BoolVar(&skipPush, "skip-push", false, "Skip pushing function to remote registry")
	upFlagset.BoolVar(&skipDeploy, "skip-deploy", false, "Skip function deployment")
	upFlagset.BoolVar(&watch, "watch", false, "Watch for file changes and trigger up")

	upCmd.Flags().AddFlagSet(upFlagset)

	build, _, _ := faasCmd.Find([]string{"build"})
	upCmd.Flags().AddFlagSet(build.Flags())

	push, _, _ := faasCmd.Find([]string{"push"})
	upCmd.Flags().AddFlagSet(push.Flags())

	deploy, _, _ := faasCmd.Find([]string{"deploy"})
	upCmd.Flags().AddFlagSet(deploy.Flags())

	faasCmd.AddCommand(upCmd)
}

// upCmd is a wrapper to the build, push and deploy commands
var upCmd = &cobra.Command{
	Use:   `up -f [YAML_FILE] [--skip-push] [--skip-deploy] [flags from build, push, deploy]`,
	Short: "Builds, pushes and deploys OpenFaaS function containers",
	Long: `Build, Push, and Deploy OpenFaaS function containers either via the
supplied YAML config using the "--yaml" flag (which may contain multiple function
definitions), or directly via flags.

The push step may be skipped by setting the --skip-push flag
and the deploy step with --skip-deploy.

Note: All flags from the build, push and deploy flags are valid and can be combined,
see the --help text for those commands for details.`,
	Example: `  faas-cli up -f myfn.yaml
faas-cli up --filter "*gif*" --secret dockerhuborg`,
	PreRunE: preRunUp,
	RunE:    upHandler,
}

func preRunUp(cmd *cobra.Command, args []string) error {
	if err := preRunBuild(cmd, args); err != nil {
		return err
	}
	if err := preRunDeploy(cmd, args); err != nil {
		return err
	}
	return nil
}

func upHandler(cmd *cobra.Command, args []string) error {

	ignoredDirs := []string{"build", ".git", "template"}
	ignoredSuffixes := []string{"~", "build", ".git", "template"}

	if watch {
		buildCount := 0

		debounced := debounce.New(500 * time.Millisecond)

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(err)
		}
		defer watcher.Close()

		done := make(chan bool)
		go func() {
			for {
				select {
				case event := <-watcher.Events:

					log.Printf("changed: %s", event)

					if skipIgnoredSuffix(event, ignoredSuffixes) {
						break
					}

					switch event.Op & fsnotify.Write {
					case fsnotify.Create, fsnotify.Write:
						debounced(func() {
							buildCount++
							if err := doUp(cmd, args); err != nil {
								log.Printf("Error detecting change: %v", err)
							}
							log.Printf("Completed Builds: %d", buildCount)
						})
					}
				case err := <-watcher.Errors:
					fmt.Println("error:", err)
				}
			}
		}()

		cwd, err := os.Getwd()
		if err != nil {
			fmt.Println("error:", err)
			return err
		}
		log.Println("Watching: ", cwd)

		watchedDirs := []string{}
		err = filepath.Walk(cwd,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if skipIgnoredDir(info, ignoredDirs) {
					return filepath.SkipDir
				}

				if info.IsDir() {
					err = watcher.Add(path)
					if err != nil {
						log.Fatal(err)
					}
					watchedDirs = append(watchedDirs, path)
				}
				return nil
			})
		if err != nil {
			log.Println(err)
		}

		<-done

	} else {
		doUp(cmd, args)
	}
	return nil
}

func doUp(cmd *cobra.Command, args []string) error {
	if err := runBuild(cmd, args); err != nil {
		return err
	}
	fmt.Println()
	if !skipPush {
		if err := runPush(cmd, args); err != nil {
			return err
		}
		fmt.Println()
	}
	if !skipDeploy {
		if err := runDeploy(cmd, args); err != nil {
			return err
		}
	}
	return nil
}

func skipIgnoredDir(info os.FileInfo, ignoredDirs []string) bool {
	if !info.IsDir() {
		return false
	}
	for _, ignoredDir := range ignoredDirs {
		if info.Name() == ignoredDir {
			return true
		}
	}
	return false
}

func skipIgnoredSuffix(event fsnotify.Event, ignoredSuffixes []string) bool {
	for _, ignoredSuffix := range ignoredSuffixes {
		if strings.HasSuffix(event.Name, ignoredSuffix) {
			return true
		}
	}
	return false
}
