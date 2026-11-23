// Copyright (c) 2023 F5 Inc. All rights reserved.

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/provider"
)

var (
	// version is set at build time via -ldflags "-X main.version=<version>".
	// It defaults to "dev" for local development and untagged builds.
	version = "dev"
)

// main starts the F5XC Terraform Provider gRPC server.
//
// Why: Terraform CLI communicates with the provider plugin over gRPC using
// Protocol Version 6. This entry point initializes command-line flags and starts
// the provider server with the build-time provider version.
//
// How: It parses the -debug flag for delve debugger attachment and invokes
// providerserver.Serve with provider.New(version).
//
// Parameters: none
//
// Returns: none (terminates via log.Fatal on startup error)
func main() {

	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address:         "registry.terraform.io/F5Networks/f5xc",
		Debug:           debug,
		ProtocolVersion: 6,
	})
	if err != nil {
		log.Fatal(err)
	}
}
