//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	types_pb "github.com/gogo/protobuf/types"
	"gopkg.volterra.us/stdlib/client/vesapi"

	ves_io_schema_api_credential "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/api_credential"
)

type customAPICall struct {
	method string
	uri    string
	rpcFQN string
	reqYml string
}

type stubCustomAPIClient struct {
	calls   []customAPICall
	handler func(method, uri, rpcFQN, reqYml string) (proto.Message, error)
}

func (s *stubCustomAPIClient) CustomAPI(ctx context.Context, method, uri, rpcFQN, reqYml string, opts ...vesapi.CallOpt) (proto.Message, error) {
	s.calls = append(s.calls, customAPICall{method: method, uri: uri, rpcFQN: rpcFQN, reqYml: reqYml})
	if s.handler == nil {
		return nil, nil
	}
	return s.handler(method, uri, rpcFQN, reqYml)
}

func TestApiCredentialOverrideUpdateHook_NoopWhenRenewEdgeNotRequested(t *testing.T) {
	testCases := []struct {
		name              string
		priorRenewTrigger bool
		planRenewTrigger  bool
	}{
		{name: "false_to_false", priorRenewTrigger: false, planRenewTrigger: false},
		{name: "true_to_true", priorRenewTrigger: true, planRenewTrigger: true},
		{name: "true_to_false", priorRenewTrigger: true, planRenewTrigger: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubCustomAPIClient{
				handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
					return nil, fmt.Errorf("unexpected CustomAPI call: %s %s %s", method, uri, rpcFQN)
				},
			}

			err := ApiCredentialOverrideUpdateHook(
				context.Background(),
				stub,
				"system",
				"cred-1",
				tc.priorRenewTrigger,
				tc.planRenewTrigger,
				10,
			)
			if err != nil {
				t.Fatalf("ApiCredentialOverrideUpdateHook returned error: %v", err)
			}
			if len(stub.calls) != 0 {
				t.Fatalf("expected no CustomAPI calls, got %d", len(stub.calls))
			}
		})
	}
}

func TestApiCredentialOverrideUpdateHook_RenewsOnFalseToTrue(t *testing.T) {
	expiredAt := time.Now().UTC().Add(-1 * time.Hour)

	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			switch rpcFQN {
			case "ves.io.schema.api_credential.CustomAPI.Get":
				return &ves_io_schema_api_credential.GetResponse{
					Object: &ves_io_schema_api_credential.Object{
						Spec: &ves_io_schema_api_credential.SpecType{
							GcSpec: &ves_io_schema_api_credential.GlobalSpecType{
								Type: ves_io_schema_api_credential.API_TOKEN,
								ExpirationTimestamp: &types_pb.Timestamp{
									Seconds: expiredAt.Unix(),
									Nanos:   int32(expiredAt.Nanosecond()),
								},
								Active: false,
							},
						},
					},
				}, nil
			case "ves.io.schema.api_credential.CustomAPI.Activate":
				return nil, nil
			case "ves.io.schema.api_credential.CustomAPI.Renew":
				return nil, nil
			default:
				return nil, fmt.Errorf("unexpected rpc %s", rpcFQN)
			}
		},
	}

	err := ApiCredentialOverrideUpdateHook(
		context.Background(),
		stub,
		"system",
		"cred-1",
		false,
		true,
		30,
	)
	if err != nil {
		t.Fatalf("ApiCredentialOverrideUpdateHook returned error: %v", err)
	}

	if len(stub.calls) != 3 {
		t.Fatalf("expected 3 CustomAPI calls (Get, Activate, Renew), got %d", len(stub.calls))
	}
	if stub.calls[0].rpcFQN != "ves.io.schema.api_credential.CustomAPI.Get" {
		t.Fatalf("first rpc = %q, want Get", stub.calls[0].rpcFQN)
	}
	if stub.calls[1].rpcFQN != "ves.io.schema.api_credential.CustomAPI.Activate" {
		t.Fatalf("second rpc = %q, want Activate", stub.calls[1].rpcFQN)
	}
	if stub.calls[2].rpcFQN != "ves.io.schema.api_credential.CustomAPI.Renew" {
		t.Fatalf("third rpc = %q, want Renew", stub.calls[2].rpcFQN)
	}
}
