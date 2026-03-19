// Copyright (c) 2023 F5 Inc. All rights reserved.

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/provider"
)

func main() {

	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New("2.0.0"), providerserver.ServeOpts{
		Address:         "registry.terraform.io/f5/f5xc",
		Debug:           debug,
		ProtocolVersion: 6,
	})
	if err != nil {
		log.Fatal(err)
	}
}
