## faas-cli
[![Go Report Card](https://goreportcard.com/badge/github.com/openfaas/faas-cli)](https://goreportcard.com/report/github.com/openfaas/faas-cli) [![Build Status](https://travis-ci.org/openfaas/faas-cli.svg?branch=master)](https://travis-ci.org/openfaas/faas-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![OpenFaaS](https://img.shields.io/badge/openfaas-serverless-blue.svg)](https://www.openfaas.com)

This is a CLI for use with [OpenFaaS](https://github.com/openfaas/faas) - a serverless functions framework for Docker & Kubernetes.

> Before using this tool please setup OpenFaaS by following instructions over on the main repo.

The CLI can be used to build and deploy functions to [OpenFaaS](https://github.com/openfaas/faas). You can build OpenFaaS functions from a set of supported language templates (such as Node.js, Python, CSharp and Ruby). That means you just write a handler file such as (handler.py/handler.js) and the CLI does the rest to create a Docker image.

Demo: [ASCII cinema](https://asciinema.org/a/141284)

### TL;DR - Introductory tutorial

[Blog: Coffee with the FaaS-CLI](https://blog.alexellis.io/quickstart-openfaas-cli/)

### FAQ

* Does the CLI hang while deploying `faas deploy`?

Checkout the [troubleshooting guide](https://github.com/openfaas/faas/blob/master/guide/troubleshooting.md)

* Can you not push your image via `faas push`?

Make sure you have specified your Docker Hub user account in your YAML file - so if your account is `alexellis2` and your image name is `demo`, then put:

```
    image: alexellis2/demo
```

* Need other help with troubleshooting?

Checkout the [troubleshooting guide](https://github.com/openfaas/faas/blob/master/guide/troubleshooting.md)

### Get started: Install the CLI

You can install the CLI with a `curl` utility script, `brew` or by downloading the binary from the releases page. Once installed you'll get the `faas-cli` command and `faas` alias.

Utility script with `curl`:

```
$ curl -sSL https://cli.openfaas.com | sudo sh
```

Non-root with curl (requires further actions as advised after downloading):

```
$ curl -sSL https://cli.openfaas.com | sh
```

Via brew:

```
$ brew install faas-cli
```

Note: The `brew` release may not run the latest minor release but is updated regularly.

In PowerShell:

```
$version = (Invoke-WebRequest "https://api.github.com/repos/openfaas/faas-cli/releases/latest" | ConvertFrom-Json)[0].tag_name
(New-Object System.Net.WebClient).DownloadFile("https://github.com/openfaas/faas-cli/releases/download/$version/faas-cli.exe", "faas-cli.exe")
```

#### Windows

The easiest way to install the faas-cli is through `scoop`:

```
scoop install faas-cli
```

Note: The `scoop` release may not run the latest minor release but is updated regularly.

#### Build from source

> the [contributing guide](CONTRIBUTING.md) has instructions for building from source and for configuring a Golang development environment.

### Run the CLI

The main commands supported by the CLI are:

* `faas-cli new` - creates a new function via a template in the current directory
* `faas-cli build` - builds Docker images from the supported language types
* `faas-cli push` - pushes Docker images into a registry
* `faas-cli deploy` - deploys the functions into a local or remote OpenFaaS gateway
* `faas-cli remove` - removes the functions from a local or remote OpenFaaS gateway
* `faas-cli invoke` - invokes the functions and reads from STDIN for the body of the request
* `faas-cli login` - stores basic auth credentials for OpenFaaS gateway (supports multiple gateways)
* `faas-cli logout` - removes basic auth credentials for a given gateway
* `faas-cli store` - allows browsing and deploying OpenFaaS store functions

The default gateway URL of `127.0.0.1:8080` can be overriden in three places including an environmental variable.

* 1st priority `--gateway` flag
* 2nd priority `--yaml` / `-f` flag or `stack.yml` if in current directory
* 3rd priority `OPENFAAS_URL` environmental variable

For Kubernetes users you may want to set this in your `.bash_rc` file:

```
export OPENFAAS_URL=127.0.0.1:31112
```

Advanced commands:

* `faas-cli template pull` - pull in templates from a remote GitHub repository [Detailed Documentation](guide/TEMPLATE.md)

Help for all of the commands supported by the CLI can be found by running:

* `faas-cli help` or `faas-cli [command] --help`

You can chose between using a [programming language template](https://github.com/openfaas/faas-cli/tree/master/template) where you only need to provide a handler file, or a Docker that you can build yourself.

#### Templates

Command: `faas-cli new FUNCTION_NAME --lang python/node/go/ruby/Dockerfile/etc`

In your YAML you can also specify `lang: node/python/go/csharp/ruby`

* Supports common languages
* Quick and easy - just write one file
* Specify depenencies on Gemfile / requirements.txt or package.json etc

* Customise the provided templates

Perhaps you need to have [`gcc` or another dependency](https://github.com/openfaas/faas-office-sample) in your Python template? That's not a problem.

You can customise the Dockerfile or code for any of the templates. Just create a new directory and copy in the templates folder from this repository. The templates in your current working directory are always used for builds.

See also: `faas-cli new --help`

**Third-party community templates**

Templates created and maintained by a third-party can be added to your local system using the `faas-cli template pull` command.

Curated language templates:

| Language  | Author   | URL  |
|---|---|---|
| PHP | @itscaro   | https://github.com/itscaro/openfaas-template-php/  |
| PHP5 | @itscaro   | https://github.com/itscaro/openfaas-template-php/  |
| F# | @hayer | https://github.com/hayer/faas-fsharp-template/ |

Read more on [community templates here](guide/TEMPLATE.md).

#### Docker image as a function

Specify `lang: Dockerfile` if you want the faas-cli to execute a build or `skip_build: true` for pre-built images.

* Ultimate versatility and control
* Package anything
* If you are using a stack file add the `skip_build: true` attribute
* Use one of the [samples as a basis](https://github.com/openfaas/faas/tree/master/sample-functions)

Read the blog post/tutorial: [Turn Any CLI into a Function with OpenFaaS](https://blog.alexellis.io/cli-functions-with-openfaas/)

### Private registries

* For Kubernetes

Create a named image pull secret and add the secret name to the `secrets` section of your YAML file or your deployment arguments with `--secret`.

Alternatively you can assign a secret to the node to allow it to pull from your private registry. In this case you do not need to assign the secret to your function.

* For Docker Swarm

For Docker Swarm use the `--send-registry-auth` flag or its shorthand `-a` which will look up your registry credentials in your local credentials store and then transmit them over the wire to the deploy command on the API Gateway. Make sure HTTPS/TLS is enabled before attempting this.

### Use a YAML stack file

A YAML stack file groups functions together and also saves on typing.

You can define individual functions or a set of of them within a YAML file. This makes the CLI easier to use and means you can use this file to deploy to your OpenFaaS instance.  By default the faas-cli will attempt to load `stack.yaml` from the current directory.

Here is an example file using the `stack.yml` file included in the repository.

```yaml
provider:
  name: faas
  gateway: http://127.0.0.1:8080

functions:
  url-ping:
    lang: python
    handler: ./sample/url-ping
    image: alexellis2/faas-urlping
```

This url-ping function is defined in the sample/url-ping folder makes use of Python. All we had to do was to write a `handler.py` file and then to list off any Python modules in `requirements.txt`.

* Build the files in the .yml file:

```
$ faas-cli build -f ./stack.yml
```

> `-f` specifies the file or URL to download your YAML file from. The long version of the `-f` flag is: `--yaml`.

You can also download over HTTP/s:

```
$ faas-cli build -f https://raw.githubusercontent.com/openfaas/faas-cli/master/stack.yml
```

Docker along with a Python template will be used to build an image named alexellis2/faas-urlping.

* Deploy your function

Now you can use the following command to deploy your function(s):

```
$ faas-cli deploy -f ./stack.yml
```

### YAML format reference

#### Secure secret management

Secrets can be used with OpenFaaS when using Docker Swarm or Kubernetes, this means your data is encrypted at rest and is less likely to be leaked during logging / stack traces than with environmental variables.

```yaml
  secrets:
    - secret-name-1
    - secret-name-2
```

Secrets should be defined in the cluster ahead of time using `docker secret create` or `kubectl`.

#### Environmental variables/configuration

You can deploy non-encrypted secrets and configuration via environmental variables set either in-line or via external (environment) files.

> Note: external files take priority over in-line environmental variables. This allows you to specify a default and then have overrides within an external file.

Priority:

* environment_file - defined in zero to many external files

```yaml
  environment_file:
    - file1.yml
    - file2.yml
```

If you specify a variable such as "access_key" in more than one `environment_file` file then the last file in the list will take priority.

Environment file format:

```yaml
environment:
  access_key: key1
  secret_key: key2
```

* Define environment in-line within the file:

Imagine you needed to define a `http_proxy` variable to operate within a corporate network:

```yaml
functions:
  url-ping:
    lang: python
    handler: ./sample/url-ping
    image: alexellis2/faas-urlping
    environment:
      http_proxy: http://proxy1.corp.com:3128
      no_proxy: http://gateway/
```

#### Constraints

Constraints work with Docker Swarm and are useful for pinning functions to certain hosts.

Here is an example of picking only Linux:

```yaml
   constraints:
     - "node.platform.os == linux"
```

Or only Windows:

```yaml
   constraints:
     - "node.platform.os == windows"
```

#### Labels

Labels can be applied through a map which may be consumed by the back-end scheduler such as Docker Swarm or Kubernetes.

For example:

```yaml
   labels:
     kafka.topic: topic1
     canary: true
```

#### Other YAML fields

The possible entries for functions are documented below:

```yaml
functions:
  deployed_function_name:
    lang: node or python (optional)
    handler: ./path/to/handler (optional)
    image: docker-image-name
    environment:
      env1: value1
      env2: "value2"
    labels:
      label1: value1
      label2: "value2"
   constraints:
     - "com.hdd == ssd"
```

Use environmental variables for setting tokens and configuration.

### Access functions with `curl`

You can initiate a HTTP POST via `curl`:

* with the `-d` flag i.e. `-d "my data here"`
* or with `--data-binary @filename.txt` to send a whole file including newlines
* if you want to pass input from STDIN then use `--data-binary @-`

```
$ curl -d '{"hello": "world"}' http://127.0.0.1:8080/function/nodejs-echo
{ nodeVersion: 'v6.9.1', input: '{"hello": "world"}' }

$ curl --data-binary @README.md http://127.0.0.1:8080/function/nodejs-echo

$ uname -a | curl http://127.0.0.1:8080/function/nodejs-echo--data-binary @-
```

> For further instructions on the manual CLI flags (without using a YAML file) read [manual_cli.md](https://github.com/openfaas/faas-cli/blob/master/MANUAL_CLI.md)

### FaaS-CLI Developers / Contributors

See [contributing guide](https://github.com/openfaas/faas-cli/blob/master/CONTRIBUTING.md).

### License

This project is part of the OpenFaaS project licensed under the MIT License.
