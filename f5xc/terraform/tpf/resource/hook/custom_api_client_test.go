//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
)

// The provider passes *client.APIClient to hooks that take a CustomAPIClient.
// This assertion fails the build if the two signatures drift apart.
var _ CustomAPIClient = (*client.APIClient)(nil)
