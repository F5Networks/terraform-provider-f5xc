#!/bin/bash
# Copyright (c) 2026 F5, Inc. All rights reserved.

# Exit immediately if a command exits with a non-zero status
set -e

# Change directory to the location of this script
cd "$(dirname "$0")"

# Run the terraform-plugin-docs generation tool
# --provider-dir: Points to the folder containing main.go
# --examples-dir: From cmd/terraform-provider-f5xc, go up 2 levels then to f5xc/terraform/tpf/examples
# --rendered-website-dir: Same logic, pointing to your custom docs folder
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate \
  --provider-name f5xc \
  --provider-dir ../../../../cmd/terraform-provider-f5xc \
  --examples-dir ../../f5xc/terraform/tpf/examples \
  --rendered-website-dir ../../f5xc/terraform/tpf/docs \
  --website-source-dir ../../f5xc/terraform/tpf/templates