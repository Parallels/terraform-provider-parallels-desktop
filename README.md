# Terraform Provider for Parallels Desktop

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT) 
[![](https://dcbadge.vercel.app/api/server/pEwZ254C3d?style=flat&theme=default)](https://discord.gg/pEwZ254C3d)
[![Release](https://github.com/Parallels/terraform-provider-parallels/actions/workflows/release.yml/badge.svg)](https://github.com/Parallels/terraform-provider-parallels/actions/workflows/release.yml)
[![Tests](https://github.com/Parallels/terraform-provider-parallels/actions/workflows/test.yml/badge.svg)](https://github.com/Parallels/terraform-provider-parallels/actions/workflows/test.yml)

<img src="https://raw.githubusercontent.com/hashicorp/terraform-website/master/public/img/logo-hashicorp.svg" width="600px">

## Maintainers

This provider plugin is maintained by Linode and the community, please check the [Code of Conduct](./CODE_OF_CONDUCT.md) if you want to participate.


## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.19

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

Check the latest examples in the terraform registry [here](https://registry.terraform.io/providers/Parallels/parallels-desktop/latest/docs).

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```
