// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build tools

package tools

//go:generate go install github.com/bflad/tfproviderlint/cmd/tfproviderlint
//go:generate go install github.com/golangci/golangci-lint/cmd/golangci-lint
//go:generate go install golang.org/x/tools/cmd/goimports
//go:generate go install mvdan.cc/gofumpt

import (
	// Documentation generation
	// side effect imports used to version go tools
	// see: https://github.com/go-modules-by-example/index/blob/master/010_tools/README.md#tools-as-dependencies
	_ "github.com/bflad/tfproviderlint/cmd/tfproviderlint"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
	_ "mvdan.cc/gofumpt"
)
