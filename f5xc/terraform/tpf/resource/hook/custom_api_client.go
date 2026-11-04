//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"gopkg.volterra.us/stdlib/client/vesapi"
)

// CustomAPIClient is the minimal interface required by lifecycle hooks to call F5XC Custom APIs.
type CustomAPIClient interface {
	CustomAPI(ctx context.Context, method, uri, rpcFQN, reqYml string, opts ...vesapi.CallOpt) (proto.Message, error)
}
