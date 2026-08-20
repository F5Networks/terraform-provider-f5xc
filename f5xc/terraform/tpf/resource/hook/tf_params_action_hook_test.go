//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	ves_io_schema_tf_params "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/views/terraform_parameters"
)

func buildTfParamsActionList(t *testing.T, values ...string) types.List {
	t.Helper()

	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}

	listValue, diags := types.ListValue(types.StringType, elements)
	if diags.HasError() {
		t.Fatalf("failed to create action list: %v", diags)
	}

	return listValue
}

func TestTfParamsActionPostCreateHook_NoWaitSkipsPolling(t *testing.T) {
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			return nil, fmt.Errorf("unexpected CustomAPI call: %s %s %s %s", method, uri, rpcFQN, reqYml)
		},
	}

	tfOutput := types.StringUnknown()
	err := TfParamsActionPostCreateHook(
		context.Background(),
		stub,
		"system",
		"aws_vpc_site",
		"site-a",
		"APPLY",
		false,
		&tfOutput,
	)
	if err != nil {
		t.Fatalf("TfParamsActionPostCreateHook returned error: %v", err)
	}
	if len(stub.calls) != 0 {
		t.Fatalf("expected no CustomAPI calls, got %d", len(stub.calls))
	}
	if !tfOutput.IsNull() {
		t.Fatalf("expected tf_output to be null when wait is disabled, got %#v", tfOutput)
	}
}

func TestTfParamsActionPostCreateHook_ApplySuccessSetsOutput(t *testing.T) {
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			return &ves_io_schema_tf_params.GetStatusResponse{
				Status: &ves_io_schema_tf_params.StatusObject{
					ApplyStatus: &ves_io_schema_tf_params.ApplyStatus{
						State:    &ves_io_schema_tf_params.ApplyStatus_ApplyState{ApplyState: ves_io_schema_tf_params.APPLIED},
						TfOutput: "site-ready",
					},
				},
			}, nil
		},
	}

	tfOutput := types.StringNull()
	err := TfParamsActionPostCreateHook(
		context.Background(),
		stub,
		"system",
		"aws_vpc_site",
		"site-a",
		"APPLY",
		true,
		&tfOutput,
	)
	if err != nil {
		t.Fatalf("TfParamsActionPostCreateHook returned error: %v", err)
	}
	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 CustomAPI call, got %d", len(stub.calls))
	}
	if tfOutput.ValueString() != "site-ready" {
		t.Fatalf("expected tf_output=site-ready, got %q", tfOutput.ValueString())
	}
}

func TestTfParamsActionOverrideUpdateHook_SkipsRerunWhenActionUnchanged(t *testing.T) {
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			switch rpcFQN {
			case tfParamsGetStatusRPCFQN:
				return &ves_io_schema_tf_params.GetStatusResponse{
					Status: &ves_io_schema_tf_params.StatusObject{
						ApplyStatus: &ves_io_schema_tf_params.ApplyStatus{
							State:    &ves_io_schema_tf_params.ApplyStatus_ApplyState{ApplyState: ves_io_schema_tf_params.APPLIED},
							TfOutput: "existing-output",
						},
					},
				}, nil
			case tfParamsActionRunRPCFQN:
				return nil, fmt.Errorf("unexpected run invocation")
			default:
				return nil, fmt.Errorf("unexpected rpc %s", rpcFQN)
			}
		},
	}

	req := &ves_io_schema_tf_params.RunRequest{
		Namespace: "system",
		ViewKind:  "aws_vpc_site",
		ViewName:  "site-a",
		Action:    ves_io_schema_tf_params.APPLY,
	}
	tfOutput := types.StringUnknown()

	err := TfParamsActionOverrideUpdateHook(
		context.Background(),
		stub,
		req,
		"aws_vpc_site",
		"site-a",
		buildTfParamsActionList(t, "APPLY"),
		true,
		&tfOutput,
	)
	if err != nil {
		t.Fatalf("TfParamsActionOverrideUpdateHook returned error: %v", err)
	}
	if len(stub.calls) != 1 {
		t.Fatalf("expected one CustomAPI call, got %d", len(stub.calls))
	}
	if stub.calls[0].rpcFQN != tfParamsGetStatusRPCFQN {
		t.Fatalf("expected first rpc %q, got %q", tfParamsGetStatusRPCFQN, stub.calls[0].rpcFQN)
	}
	if tfOutput.ValueString() != "existing-output" {
		t.Fatalf("expected tf_output=existing-output, got %q", tfOutput.ValueString())
	}
}

func TestTfParamsActionOverrideUpdateHook_RerunsWhenActionChanges(t *testing.T) {
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			switch rpcFQN {
			case tfParamsActionRunRPCFQN:
				return &ves_io_schema_tf_params.RunResponse{}, nil
			case tfParamsGetStatusRPCFQN:
				return nil, fmt.Errorf("unexpected status invocation")
			default:
				return nil, fmt.Errorf("unexpected rpc %s", rpcFQN)
			}
		},
	}

	req := &ves_io_schema_tf_params.RunRequest{
		Namespace: "system",
		ViewKind:  "aws_vpc_site",
		ViewName:  "site-a",
		Action:    ves_io_schema_tf_params.APPLY,
	}
	tfOutput := types.StringUnknown()

	err := TfParamsActionOverrideUpdateHook(
		context.Background(),
		stub,
		req,
		"aws_vpc_site",
		"site-a",
		buildTfParamsActionList(t, "PLAN"),
		false,
		&tfOutput,
	)
	if err != nil {
		t.Fatalf("TfParamsActionOverrideUpdateHook returned error: %v", err)
	}
	if len(stub.calls) != 1 {
		t.Fatalf("expected one CustomAPI call, got %d", len(stub.calls))
	}
	if stub.calls[0].rpcFQN != tfParamsActionRunRPCFQN {
		t.Fatalf("expected run rpc %q, got %q", tfParamsActionRunRPCFQN, stub.calls[0].rpcFQN)
	}
	if !tfOutput.IsNull() {
		t.Fatalf("expected tf_output to be null when wait is disabled, got %#v", tfOutput)
	}
}

func TestTfParamsActionOverrideDeleteHook_UsesDestroyAction(t *testing.T) {
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			if rpcFQN != tfParamsActionRunRPCFQN {
				return nil, fmt.Errorf("unexpected rpc %s", rpcFQN)
			}
			if !strings.Contains(reqYml, "action: DESTROY") {
				return nil, fmt.Errorf("destroy action missing from request: %s", reqYml)
			}
			return &ves_io_schema_tf_params.RunResponse{}, nil
		},
	}

	err := TfParamsActionOverrideDeleteHook(
		context.Background(),
		stub,
		"",
		"aws_vpc_site",
		"site-a",
		false,
	)
	if err != nil {
		t.Fatalf("TfParamsActionOverrideDeleteHook returned error: %v", err)
	}
	if len(stub.calls) != 1 {
		t.Fatalf("expected one CustomAPI call, got %d", len(stub.calls))
	}
	if stub.calls[0].uri != "/public/namespaces/system/terraform/aws_vpc_site/site-a/run" {
		t.Fatalf("unexpected uri %q", stub.calls[0].uri)
	}
}

func TestTfParamsActionOverrideDeleteHook_WaitTreats404AsSuccess(t *testing.T) {
	call := 0
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			call++
			switch call {
			case 1:
				if rpcFQN != tfParamsActionRunRPCFQN {
					return nil, fmt.Errorf("expected run rpc, got %s", rpcFQN)
				}
				return &ves_io_schema_tf_params.RunResponse{}, nil
			case 2:
				if rpcFQN != tfParamsGetStatusRPCFQN {
					return nil, fmt.Errorf("expected status rpc, got %s", rpcFQN)
				}
				return nil, fmt.Errorf("status code 404")
			default:
				return nil, fmt.Errorf("unexpected extra call %d", call)
			}
		},
	}

	err := TfParamsActionOverrideDeleteHook(
		context.Background(),
		stub,
		"system",
		"aws_vpc_site",
		"site-a",
		true,
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(stub.calls) != 2 {
		t.Fatalf("expected two CustomAPI calls, got %d", len(stub.calls))
	}
}

func TestDefaultTfParamsPollConfig_ApplyUsesLegacyCadence(t *testing.T) {
	cfg := defaultTfParamsPollConfig("apply")

	if cfg.maxAttempts != tfParamsApplyDestroyPollMaxAttempts {
		t.Fatalf("expected apply maxAttempts=%d, got %d", tfParamsApplyDestroyPollMaxAttempts, cfg.maxAttempts)
	}
	if cfg.pollInterval != 30*time.Second {
		t.Fatalf("expected apply pollInterval=30s, got %s", cfg.pollInterval)
	}
}

func TestDefaultTfParamsPollConfig_DestroyUsesLegacyCadence(t *testing.T) {
	cfg := defaultTfParamsPollConfig("destroy")

	if cfg.maxAttempts != tfParamsApplyDestroyPollMaxAttempts {
		t.Fatalf("expected destroy maxAttempts=%d, got %d", tfParamsApplyDestroyPollMaxAttempts, cfg.maxAttempts)
	}
	if cfg.pollInterval != 30*time.Second {
		t.Fatalf("expected destroy pollInterval=30s, got %s", cfg.pollInterval)
	}
}

func TestDefaultTfParamsPollConfig_PlanUsesLegacyCadence(t *testing.T) {
	cfg := defaultTfParamsPollConfig("plan")

	if cfg.maxAttempts != tfParamsPlanPollMaxAttempts {
		t.Fatalf("expected plan maxAttempts=%d, got %d", tfParamsPlanPollMaxAttempts, cfg.maxAttempts)
	}
	if cfg.pollInterval != 10*time.Second {
		t.Fatalf("expected plan pollInterval=10s, got %s", cfg.pollInterval)
	}
}
