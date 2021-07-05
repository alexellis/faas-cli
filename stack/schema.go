// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package stack

// Provider for the FaaS set of functions.
type Provider struct {
	Name       string `yaml:"name"`
	GatewayURL string `yaml:"gateway"`
}

// Function as deployed or built on FaaS
type Function struct {
	// Name of deployed function
	Name string `yaml:"-"`

	Language string `yaml:"lang"`

	// Handler Local folder to use for function
	Handler string `yaml:"handler"`

	// Image Docker image name
	Image string `yaml:"image"`

	FProcess string `yaml:"fprocess"`

	Environment map[string]string `yaml:"environment"`

	// Secrets list of secrets to be made available to function
	Secrets []string `yaml:"secrets,omitempty"`

	SkipBuild bool `yaml:"skip_build,omitempty"`

	Constraints *[]string `yaml:"constraints,omitempty"`

	// EnvironmentFile is a list of files to import and override environmental variables.
	// These are overriden in order.
	EnvironmentFile []string `yaml:"environment_file,omitempty"`

	Labels *map[string]string `yaml:"labels,omitempty"`

	// Limits for function
	Limits *FunctionResources `yaml:"limits,omitempty"`

	// Requests of resources requested by function
	Requests *FunctionResources `yaml:"requests,omitempty"`

	// ReadOnlyRootFilesystem is used to set the container filesystem to read-only
	ReadOnlyRootFilesystem bool `yaml:"readonly_root_filesystem,omitempty"`

	// BuildOptions to determine native packages
	BuildOptions []string `yaml:"build_options,omitempty"`

	// Annotations
	Annotations *map[string]string `yaml:"annotations,omitempty"`

	// Namespace of the function
	Namespace string `yaml:"namespace,omitempty"`

	// BuildArgs for providing build-args
	BuildArgs map[string]string `yaml:"build_args,omitempty"`

	// IgnorePaths are relative files or directories that will not be copied to the
	// build folder e.g. node_modules
	IgnorePaths []string `yaml:"ignore_paths,omitempty"`

	// Platforms for use with buildx and faas-cli publish
	Platforms string `yaml:"platforms,omitempty"`
}

// Configuration for the stack.yml file
type Configuration struct {
	StackConfig StackConfiguration `yaml:"configuration"`
}

// StackConfiguration for the overall stack.yml
type StackConfiguration struct {
	TemplateConfigs []TemplateSource `yaml:"templates"`

	// CopyExtraPaths specifies additional paths (relative to the stack file) that will be copied
	// into the functions build context, e.g. specifying `"common"` will look for and copy the
	// "common/" folder of file in the same root as the stack file.  All paths must be contained
	// within the project root defined by the location of the stack file.
	//
	// The yaml uses the shorter name `copy` to make it easier for developers to read and use
	CopyExtraPaths []string `yaml:"copy"`
}

// TemplateSource for build templates
type TemplateSource struct {
	Name   string `yaml:"name"`
	Source string `yaml:"source,omitempty"`
}

// FunctionResources Memory and CPU
type FunctionResources struct {
	Memory string `yaml:"memory"`
	CPU    string `yaml:"cpu"`
}

// EnvironmentFile represents external file for environment data
type EnvironmentFile struct {
	Environment map[string]string `yaml:"environment"`
}

// Services root level YAML file to define FaaS function-set
type Services struct {
	Version            string              `yaml:"version,omitempty"`
	Functions          map[string]Function `yaml:"functions,omitempty"`
	Provider           Provider            `yaml:"provider,omitempty"`
	StackConfiguration StackConfiguration  `yaml:"configuration,omitempty"`
}

// LanguageTemplate read from template.yml within root of a language template folder
type LanguageTemplate struct {
	Language string `yaml:"language,omitempty"`
	FProcess string `yaml:"fprocess,omitempty"`

	BuildOptions []BuildOption `yaml:"build_options,omitempty"`
	// WelcomeMessage is printed to the user after generating a function
	WelcomeMessage string `yaml:"welcome_message,omitempty"`
	// HandlerFolder to copy the function code into
	HandlerFolder string `yaml:"handler_folder,omitempty"`
}

// BuildOption a named build option for one or more packages
type BuildOption struct {
	Name     string   `yaml:"name"`
	Packages []string `yaml:"packages"`
}
