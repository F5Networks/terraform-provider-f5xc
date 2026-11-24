//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func tfParamsRunRequest(action ves_io_schema_tf_params.RunAction) *ves_io_schema_tf_params.RunRequest {
	return &ves_io_schema_tf_params.RunRequest{
		Namespace: "system",
		ViewKind:  "aws_vpc_site",
		ViewName:  "site-a",
		Action:    action,
	}
}

func planStatusResp(state ves_io_schema_tf_params.PlanStageState) *ves_io_schema_tf_params.GetStatusResponse {
	return &ves_io_schema_tf_params.GetStatusResponse{
		Status: &ves_io_schema_tf_params.StatusObject{
			PlanStatus: &ves_io_schema_tf_params.PlanStatus{
				State: &ves_io_schema_tf_params.PlanStatus_PlanState{PlanState: state},
			},
		},
	}
}

// noSleepPollConfig returns a poll config that never blocks and fails the
// sleep after failAfter calls, when failAfter is positive.
func noSleepPollConfig(maxAttempts, failAfter int) (tfParamsPollConfig, *int) {
	calls := 0
	return tfParamsPollConfig{
		maxAttempts:  maxAttempts,
		pollInterval: time.Millisecond,
		sleep: func(context.Context, time.Duration) error {
			calls++
			if failAfter > 0 && calls >= failAfter {
				return context.Canceled
			}
			return nil
		},
	}, &calls
}

// The update hook rejects a nil request before any API call.
func TestTfParamsActionOverrideUpdateHook_NilRequest(t *testing.T) {
	stub := &stubCustomAPIClient{}
	err := TfParamsActionOverrideUpdateHook(context.Background(), stub, nil, "", "", types.ListNull(types.StringType), false, nil)
	require.ErrorContains(t, err, "run request is nil")
	assert.Empty(t, stub.calls)
}

// A prior action list that cannot decode into strings returns an error.
func TestTfParamsActionOverrideUpdateHook_BadPriorAction(t *testing.T) {
	badList, diags := types.ListValue(types.Int64Type, []attr.Value{types.Int64Value(1)})
	require.False(t, diags.HasError())

	err := TfParamsActionOverrideUpdateHook(context.Background(), &stubCustomAPIClient{},
		tfParamsRunRequest(ves_io_schema_tf_params.APPLY), "aws_vpc_site", "site-a", badList, false, nil)
	require.ErrorContains(t, err, "failed to decode action list")
}

// A run failure on update is returned to the caller.
func TestTfParamsActionOverrideUpdateHook_RunError(t *testing.T) {
	stub := &stubCustomAPIClient{handler: func(_, _, _, _ string) (proto.Message, error) {
		return nil, errors.New("boom")
	}}
	err := TfParamsActionOverrideUpdateHook(context.Background(), stub,
		tfParamsRunRequest(ves_io_schema_tf_params.APPLY), "other_kind", "site-a", types.ListNull(types.StringType), false, nil)
	require.ErrorContains(t, err, "custom API run failed")
}

// A run failure on delete is returned before any wait.
func TestTfParamsActionOverrideDeleteHook_RunError(t *testing.T) {
	stub := &stubCustomAPIClient{handler: func(_, _, _, _ string) (proto.Message, error) {
		return nil, errors.New("boom")
	}}
	err := TfParamsActionOverrideDeleteHook(context.Background(), stub, "system", "aws_vpc_site", "site-a", true)
	require.ErrorContains(t, err, "custom API run failed")
	assert.Len(t, stub.calls, 1)
}

// The action runs again when the view, the prior action presence, or the action changes.
func TestShouldRunTfParamsActionOnUpdate(t *testing.T) {
	cases := []struct {
		name                   string
		pKind, pName, pAction  string
		hasPrior               bool
		kind, viewName, action string
		want                   bool
	}{
		{"kind changes", "a", "n", "APPLY", true, "b", "n", "APPLY", true},
		{"name changes", "a", "n", "APPLY", true, "a", "m", "APPLY", true},
		{"no prior action", "a", "n", "", false, "a", "n", "APPLY", true},
		{"action changes", "a", "n", "PLAN", true, "a", "n", "APPLY", true},
		{"unchanged ignores case and space", " A ", "n", "apply", true, "a", " N", "APPLY", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldRunTfParamsActionOnUpdate(tc.pKind, tc.pName, tc.pAction, tc.hasPrior, tc.kind, tc.viewName, tc.action)
			assert.Equal(t, tc.want, got)
		})
	}
}

// Null, unknown, empty, and null-element lists carry no prior action.
func TestExtractTfParamsActionValue_NoValue(t *testing.T) {
	empty, diags := types.ListValue(types.StringType, []attr.Value{})
	require.False(t, diags.HasError())
	nullElem, diags := types.ListValue(types.StringType, []attr.Value{types.StringNull()})
	require.False(t, diags.HasError())

	for name, l := range map[string]types.List{
		"null":         types.ListNull(types.StringType),
		"unknown":      types.ListUnknown(types.StringType),
		"empty":        empty,
		"null element": nullElem,
	} {
		t.Run(name, func(t *testing.T) {
			v, ok, err := extractTfParamsActionValue(context.Background(), l)
			require.NoError(t, err)
			assert.False(t, ok)
			assert.Empty(t, v)
		})
	}
}

// runTfParamsAction rejects a nil request.
func TestRunTfParamsAction_NilRequest(t *testing.T) {
	require.ErrorContains(t, runTfParamsAction(context.Background(), &stubCustomAPIClient{}, nil), "run request is nil")
}

// The wait flow requires a non-blank action.
func TestWaitForTfParamsAction_BlankAction(t *testing.T) {
	out := types.StringValue("stale")
	err := waitForTfParamsAction(context.Background(), &stubCustomAPIClient{}, "", "k", "n", " ", true, &out)
	require.ErrorContains(t, err, "action is required for wait flow")
	assert.True(t, out.IsNull(), "tf_output must reset before the wait")
}

// A wait failure on update is returned to the caller.
func TestWaitForTfParamsAction_ErrorPropagates(t *testing.T) {
	stub := &stubCustomAPIClient{handler: func(_, _, _, _ string) (proto.Message, error) {
		return planStatusResp(ves_io_schema_tf_params.PLAN_ERRORED), nil
	}}
	err := TfParamsActionPostCreateHook(context.Background(), stub, "", "k", "n", "PLAN", true, nil)
	require.ErrorContains(t, err, "terraform plan failed")
}

// sleepWithContext returns nil after the duration and the context error on cancel.
func TestSleepWithContext(t *testing.T) {
	require.NoError(t, sleepWithContext(context.Background(), time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, sleepWithContext(ctx, time.Hour), context.Canceled)
}

// The poll loop retries status errors and nil status objects, then completes.
func TestWaitForTfParamsActionToComplete_RetriesThenSucceeds(t *testing.T) {
	call := 0
	stub := &stubCustomAPIClient{handler: func(_, _, _, _ string) (proto.Message, error) {
		call++
		switch call {
		case 1:
			return nil, errors.New("transient")
		case 2:
			return &ves_io_schema_tf_params.GetStatusResponse{}, nil
		case 3:
			return planStatusResp(ves_io_schema_tf_params.PLANNING), nil
		default:
			return planStatusResp(ves_io_schema_tf_params.HAS_CHANGES), nil
		}
	}}
	cfg, sleeps := noSleepPollConfig(10, 0)

	out, err := waitForTfParamsActionToComplete(context.Background(), stub, "plan", "system", "k", "n", cfg)
	require.NoError(t, err)
	assert.Empty(t, out)
	assert.Equal(t, 3, *sleeps)
}

// Each sleep failure in the poll loop stops the wait with a context error.
func TestWaitForTfParamsActionToComplete_SleepErrors(t *testing.T) {
	cases := map[string]struct {
		resp    proto.Message
		err     error
		wantMsg string
	}{
		"after status error": {err: errors.New("transient"), wantMsg: "after status error"},
		"nil status object":  {resp: &ves_io_schema_tf_params.GetStatusResponse{}, wantMsg: "waiting for status object"},
		"while polling":      {resp: planStatusResp(ves_io_schema_tf_params.PLAN_QUEUED), wantMsg: "while polling"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			stub := &stubCustomAPIClient{handler: func(_, _, _, _ string) (proto.Message, error) {
				return tc.resp, tc.err
			}}
			cfg, _ := noSleepPollConfig(5, 1)
			_, err := waitForTfParamsActionToComplete(context.Background(), stub, "plan", "system", "k", "n", cfg)
			require.ErrorContains(t, err, tc.wantMsg)
			require.ErrorIs(t, err, context.Canceled)
		})
	}
}

// An unsupported action and an apply failure stop the poll loop with an error.
func TestWaitForTfParamsActionToComplete_Errors(t *testing.T) {
	applyErr := &ves_io_schema_tf_params.GetStatusResponse{Status: &ves_io_schema_tf_params.StatusObject{
		ApplyStatus: &ves_io_schema_tf_params.ApplyStatus{
			State:       &ves_io_schema_tf_params.ApplyStatus_ApplyState{ApplyState: ves_io_schema_tf_params.APPLY_ERRORED},
			ErrorOutput: "bad",
		},
	}}
	stub := &stubCustomAPIClient{handler: func(_, _, _, _ string) (proto.Message, error) { return applyErr, nil }}
	cfg, _ := noSleepPollConfig(3, 0)

	_, err := waitForTfParamsActionToComplete(context.Background(), stub, "upgrade", "system", "k", "n", cfg)
	require.ErrorContains(t, err, `unsupported action "upgrade"`)

	_, err = waitForTfParamsActionToComplete(context.Background(), stub, "apply", "system", "k", "n", cfg)
	require.ErrorContains(t, err, "terraform apply failed: bad")
}

// The poll loop times out after maxAttempts when the action never completes.
func TestWaitForTfParamsActionToComplete_Timeout(t *testing.T) {
	stub := &stubCustomAPIClient{handler: func(_, _, _, _ string) (proto.Message, error) {
		return planStatusResp(ves_io_schema_tf_params.PLANNING), nil
	}}
	cfg, _ := noSleepPollConfig(2, 0)
	_, err := waitForTfParamsActionToComplete(context.Background(), stub, "plan", "system", "k", "n", cfg)
	require.ErrorContains(t, err, "timed out")
	assert.Len(t, stub.calls, 2)
}

// The plan status maps each plan and infra state to pending, done, or failed.
func TestEvaluateTfParamsPlanStatus(t *testing.T) {
	plan := func(s ves_io_schema_tf_params.PlanStageState) *ves_io_schema_tf_params.PlanStatus {
		return &ves_io_schema_tf_params.PlanStatus{State: &ves_io_schema_tf_params.PlanStatus_PlanState{PlanState: s}, ErrorOutput: "e"}
	}
	infra := func(s ves_io_schema_tf_params.InfraState) *ves_io_schema_tf_params.PlanStatus {
		return &ves_io_schema_tf_params.PlanStatus{State: &ves_io_schema_tf_params.PlanStatus_InfraState{InfraState: s}}
	}
	cases := []struct {
		name    string
		in      *ves_io_schema_tf_params.PlanStatus
		done    bool
		wantErr string
	}{
		{"nil", nil, false, ""},
		{"no state", &ves_io_schema_tf_params.PlanStatus{}, false, ""},
		{"planning", plan(ves_io_schema_tf_params.PLANNING), false, ""},
		{"queued", plan(ves_io_schema_tf_params.PLAN_QUEUED), false, ""},
		{"errored", plan(ves_io_schema_tf_params.PLAN_ERRORED), false, "terraform plan failed: e"},
		{"init errored", plan(ves_io_schema_tf_params.PLAN_INIT_ERRORED), false, "terraform plan failed"},
		{"has changes", plan(ves_io_schema_tf_params.HAS_CHANGES), true, ""},
		{"infra provisioning", infra(ves_io_schema_tf_params.PROVISIONING), false, ""},
		{"infra errored", infra(ves_io_schema_tf_params.ERRORED), false, "infra state failed: ERRORED"},
		{"infra timed out", infra(ves_io_schema_tf_params.TIMED_OUT), false, "infra state failed: TIMED_OUT"},
		{"infra provisioned", infra(ves_io_schema_tf_params.PROVISIONED), true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			done, err := evaluateTfParamsPlanStatus(tc.in)
			assert.Equal(t, tc.done, done)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}

// The apply status maps apply, destroy, and infra states to pending, done, or failed.
// Only a completed apply returns tf_output.
func TestEvaluateTfParamsApplyStatus(t *testing.T) {
	apply := func(s ves_io_schema_tf_params.ApplyStageState) *ves_io_schema_tf_params.ApplyStatus {
		return &ves_io_schema_tf_params.ApplyStatus{State: &ves_io_schema_tf_params.ApplyStatus_ApplyState{ApplyState: s}, TfOutput: "out", ErrorOutput: "e"}
	}
	destroy := func(s ves_io_schema_tf_params.DestroyStageState) *ves_io_schema_tf_params.ApplyStatus {
		return &ves_io_schema_tf_params.ApplyStatus{State: &ves_io_schema_tf_params.ApplyStatus_DestroyState{DestroyState: s}, ErrorOutput: "e"}
	}
	infra := func(s ves_io_schema_tf_params.InfraState) *ves_io_schema_tf_params.ApplyStatus {
		return &ves_io_schema_tf_params.ApplyStatus{State: &ves_io_schema_tf_params.ApplyStatus_InfraState{InfraState: s}}
	}
	cases := []struct {
		name    string
		action  string
		in      *ves_io_schema_tf_params.ApplyStatus
		done    bool
		out     string
		wantErr string
	}{
		{"nil", "apply", nil, false, "", ""},
		{"no state", "apply", &ves_io_schema_tf_params.ApplyStatus{}, false, "", ""},
		{"apply planning", "apply", apply(ves_io_schema_tf_params.APPLY_PLANNING), false, "", ""},
		{"applying", "apply", apply(ves_io_schema_tf_params.APPLYING), false, "", ""},
		{"apply queued", "apply", apply(ves_io_schema_tf_params.APPLY_QUEUED), false, "", ""},
		{"apply errored", "apply", apply(ves_io_schema_tf_params.APPLY_ERRORED), false, "", "terraform apply failed: e"},
		{"apply init errored", "apply", apply(ves_io_schema_tf_params.APPLY_INIT_ERRORED), false, "", "terraform apply failed"},
		{"apply plan errored", "apply", apply(ves_io_schema_tf_params.APPLY_PLAN_ERRORED), false, "", "terraform apply failed"},
		{"applied returns output", "apply", apply(ves_io_schema_tf_params.APPLIED), true, "out", ""},
		{"applied on destroy drops output", "destroy", apply(ves_io_schema_tf_params.APPLIED), true, "", ""},
		{"destroying", "destroy", destroy(ves_io_schema_tf_params.DESTROYING), false, "", ""},
		{"destroy queued", "destroy", destroy(ves_io_schema_tf_params.DESTROY_QUEUED), false, "", ""},
		{"destroy errored", "destroy", destroy(ves_io_schema_tf_params.DESTROY_ERRORED), false, "", "terraform destroy failed: e"},
		{"destroyed", "destroy", destroy(ves_io_schema_tf_params.DESTROYED), true, "", ""},
		{"infra provisioning", "apply", infra(ves_io_schema_tf_params.PROVISIONING), false, "", ""},
		{"infra errored", "apply", infra(ves_io_schema_tf_params.ERRORED), false, "", "infra state failed: ERRORED"},
		{"infra timed out", "apply", infra(ves_io_schema_tf_params.TIMED_OUT), false, "", "infra state failed: TIMED_OUT"},
		{"infra provisioned", "apply", infra(ves_io_schema_tf_params.PROVISIONED), true, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			done, out, err := evaluateTfParamsApplyStatus(tc.action, tc.in)
			assert.Equal(t, tc.done, done)
			assert.Equal(t, tc.out, out)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}

// A status response of the wrong type is an error.
func TestGetTfParamsTerraformStatus_UnexpectedType(t *testing.T) {
	stub := &stubCustomAPIClient{handler: func(_, _, _, _ string) (proto.Message, error) {
		return &ves_io_schema_tf_params.RunResponse{}, nil
	}}
	_, err := getTfParamsTerraformStatus(context.Background(), stub, "", "k", "n")
	require.ErrorContains(t, err, "unexpected response type")
	assert.True(t, strings.HasPrefix(stub.calls[0].uri, fmt.Sprintf(tfParamsGetStatusURI, "system", "k", "n")))
}
