//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.volterra.us/stdlib/codec"

	ves_io_schema_tf_params "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/views/terraform_parameters"
)

const (
	tfParamsActionRunRPCFQN = "ves.io.schema.views.terraform_parameters.CustomActionAPI.Run"
	tfParamsActionRunURI    = "/public/namespaces/%s/terraform/%s/%s/run"

	tfParamsGetStatusRPCFQN = "ves.io.schema.views.terraform_parameters.CustomAPI.GetStatus"
	tfParamsGetStatusURI    = "/public/namespaces/%s/terraform_parameters/%s/%s/status"

	tfParamsActionDefaultNamespace = "system"

	tfParamsPlanPollMaxAttempts         = 30
	tfParamsApplyDestroyPollMaxAttempts = 30
	tfParamsPlanPollInterval            = 10 * time.Second
	tfParamsApplyDestroyPollInterval    = 30 * time.Second
)

type tfParamsPollConfig struct {
	maxAttempts  int
	pollInterval time.Duration
	sleep        func(context.Context, time.Duration) error
}

// TfParamsActionPostCreateHook waits for tf_params_action completion after the
// generated Create call has already submitted the backend Run request.
func TfParamsActionPostCreateHook(
	ctx context.Context,
	apiClient CustomAPIClient,
	namespace string,
	viewKind string,
	viewName string,
	action string,
	waitForAction bool,
	tfOutput *types.String,
) error {
	return waitForTfParamsAction(ctx, apiClient, normalizeTfParamsNamespace(namespace), viewKind, viewName, action, waitForAction, tfOutput)
}

// TfParamsActionOverrideUpdateHook runs guarded update behavior for
// tf_params_action and optionally waits for completion.
func TfParamsActionOverrideUpdateHook(
	ctx context.Context,
	apiClient CustomAPIClient,
	req *ves_io_schema_tf_params.RunRequest,
	priorViewKind string,
	priorViewName string,
	priorAction types.List,
	waitForAction bool,
	tfOutput *types.String,
) error {
	if req == nil {
		return fmt.Errorf("TfParamsActionOverrideUpdateHook: run request is nil")
	}

	req.Namespace = normalizeTfParamsNamespace(req.GetNamespace())
	action := strings.TrimSpace(req.GetAction().String())
	if action == "" {
		return fmt.Errorf("TfParamsActionOverrideUpdateHook: action is required")
	}

	priorActionValue, hasPriorAction, err := extractTfParamsActionValue(ctx, priorAction)
	if err != nil {
		return err
	}

	if shouldRunTfParamsActionOnUpdate(priorViewKind, priorViewName, priorActionValue, hasPriorAction, req.GetViewKind(), req.GetViewName(), action) {
		if err := runTfParamsAction(ctx, apiClient, req); err != nil {
			return err
		}
	}

	return waitForTfParamsAction(ctx, apiClient, req.GetNamespace(), req.GetViewKind(), req.GetViewName(), action, waitForAction, tfOutput)
}

// TfParamsActionOverrideDeleteHook executes backend DESTROY action for
// tf_params_action and optionally waits for completion.
func TfParamsActionOverrideDeleteHook(
	ctx context.Context,
	apiClient CustomAPIClient,
	namespace string,
	viewKind string,
	viewName string,
	waitForAction bool,
) error {
	req := &ves_io_schema_tf_params.RunRequest{
		Namespace: normalizeTfParamsNamespace(namespace),
		ViewKind:  viewKind,
		ViewName:  viewName,
		Action:    ves_io_schema_tf_params.DESTROY,
	}

	if err := runTfParamsAction(ctx, apiClient, req); err != nil {
		return err
	}

	return waitForTfParamsAction(ctx, apiClient, req.GetNamespace(), req.GetViewKind(), req.GetViewName(), req.GetAction().String(), waitForAction, nil)
}

func normalizeTfParamsNamespace(namespace string) string {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return tfParamsActionDefaultNamespace
	}

	return namespace
}

func shouldRunTfParamsActionOnUpdate(
	priorViewKind string,
	priorViewName string,
	priorAction string,
	hasPriorAction bool,
	plannedViewKind string,
	plannedViewName string,
	plannedAction string,
) bool {
	if !strings.EqualFold(strings.TrimSpace(priorViewKind), strings.TrimSpace(plannedViewKind)) {
		return true
	}

	if !strings.EqualFold(strings.TrimSpace(priorViewName), strings.TrimSpace(plannedViewName)) {
		return true
	}

	if !hasPriorAction {
		return true
	}

	return !strings.EqualFold(strings.TrimSpace(priorAction), strings.TrimSpace(plannedAction))
}

func extractTfParamsActionValue(ctx context.Context, actionList types.List) (string, bool, error) {
	if actionList.IsNull() || actionList.IsUnknown() {
		return "", false, nil
	}

	var actionElements []types.String
	diags := actionList.ElementsAs(ctx, &actionElements, false)
	if diags.HasError() {
		return "", false, fmt.Errorf("extractTfParamsActionValue: failed to decode action list: %s", diags.Errors()[0].Detail())
	}

	if len(actionElements) == 0 {
		return "", false, nil
	}

	actionElement := actionElements[0]
	if actionElement.IsNull() || actionElement.IsUnknown() {
		return "", false, nil
	}

	return strings.TrimSpace(actionElement.ValueString()), true, nil
}

func runTfParamsAction(
	ctx context.Context,
	apiClient CustomAPIClient,
	req *ves_io_schema_tf_params.RunRequest,
) error {
	if req == nil {
		return fmt.Errorf("runTfParamsAction: run request is nil")
	}

	req.Namespace = normalizeTfParamsNamespace(req.GetNamespace())

	yamlReq, err := codec.ToYAML(req)
	if err != nil {
		return fmt.Errorf("runTfParamsAction: failed to marshal RunRequest: %w", err)
	}

	_, err = apiClient.CustomAPI(
		ctx,
		http.MethodPost,
		fmt.Sprintf(tfParamsActionRunURI, req.GetNamespace(), req.GetViewKind(), req.GetViewName()),
		tfParamsActionRunRPCFQN,
		yamlReq,
	)
	if err != nil {
		return fmt.Errorf("runTfParamsAction: custom API run failed: %w", err)
	}

	return nil
}

func waitForTfParamsAction(
	ctx context.Context,
	apiClient CustomAPIClient,
	namespace string,
	viewKind string,
	viewName string,
	action string,
	waitForAction bool,
	tfOutput *types.String,
) error {
	if tfOutput != nil {
		*tfOutput = types.StringNull()
	}

	if !waitForAction {
		return nil
	}

	normalizedAction := strings.ToLower(strings.TrimSpace(action))
	if normalizedAction == "" {
		return fmt.Errorf("waitForTfParamsAction: action is required for wait flow")
	}

	output, err := waitForTfParamsActionToComplete(
		ctx,
		apiClient,
		normalizedAction,
		normalizeTfParamsNamespace(namespace),
		viewKind,
		viewName,
		defaultTfParamsPollConfig(normalizedAction),
	)
	if err != nil {
		return err
	}

	if tfOutput != nil && output != "" {
		*tfOutput = types.StringValue(output)
	}

	return nil
}

func defaultTfParamsPollConfig(action string) tfParamsPollConfig {
	maxAttempts := tfParamsPlanPollMaxAttempts
	pollInterval := tfParamsPlanPollInterval

	if action == "apply" || action == "destroy" {
		maxAttempts = tfParamsApplyDestroyPollMaxAttempts
		pollInterval = tfParamsApplyDestroyPollInterval
	}

	return tfParamsPollConfig{
		maxAttempts:  maxAttempts,
		pollInterval: pollInterval,
		sleep:        sleepWithContext,
	}
}

func sleepWithContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func waitForTfParamsActionToComplete(
	ctx context.Context,
	apiClient CustomAPIClient,
	action string,
	namespace string,
	viewKind string,
	viewName string,
	cfg tfParamsPollConfig,
) (string, error) {
	for i := 0; i < cfg.maxAttempts; i++ {
		statusResp, err := getTfParamsTerraformStatus(ctx, apiClient, namespace, viewKind, viewName)
		if err != nil {
			if action == "destroy" && strings.Contains(err.Error(), "status code 404") {
				return "", nil
			}
			if errSleep := cfg.sleep(ctx, cfg.pollInterval); errSleep != nil {
				return "", fmt.Errorf("waitForTfParamsActionToComplete: context done while waiting after status error: %w", errSleep)
			}
			continue
		}

		status := statusResp.GetStatus()
		if status == nil {
			if errSleep := cfg.sleep(ctx, cfg.pollInterval); errSleep != nil {
				return "", fmt.Errorf("waitForTfParamsActionToComplete: context done while waiting for status object: %w", errSleep)
			}
			continue
		}

		switch action {
		case "plan":
			done, errPlan := evaluateTfParamsPlanStatus(status.GetPlanStatus())
			if errPlan != nil {
				return "", errPlan
			}
			if done {
				return "", nil
			}
		case "apply", "destroy":
			done, output, errApply := evaluateTfParamsApplyStatus(action, status.GetApplyStatus())
			if errApply != nil {
				return "", errApply
			}
			if done {
				return output, nil
			}
		default:
			return "", fmt.Errorf("waitForTfParamsActionToComplete: unsupported action %q", action)
		}

		if errSleep := cfg.sleep(ctx, cfg.pollInterval); errSleep != nil {
			return "", fmt.Errorf("waitForTfParamsActionToComplete: context done while polling: %w", errSleep)
		}
	}

	return "", fmt.Errorf("waitForTfParamsActionToComplete: timed out waiting for action %q on %s/%s", action, viewKind, viewName)
}

func evaluateTfParamsPlanStatus(planStatus *ves_io_schema_tf_params.PlanStatus) (bool, error) {
	if planStatus == nil {
		return false, nil
	}

	switch st := planStatus.GetState().(type) {
	case *ves_io_schema_tf_params.PlanStatus_PlanState:
		planState := st.PlanState
		if planState == ves_io_schema_tf_params.PLANNING || planState == ves_io_schema_tf_params.PLAN_QUEUED {
			return false, nil
		}
		if planState == ves_io_schema_tf_params.PLAN_ERRORED || planState == ves_io_schema_tf_params.PLAN_INIT_ERRORED {
			return false, fmt.Errorf("evaluateTfParamsPlanStatus: terraform plan failed: %s", planStatus.GetErrorOutput())
		}
		return true, nil
	case *ves_io_schema_tf_params.PlanStatus_InfraState:
		infraState := st.InfraState
		if infraState == ves_io_schema_tf_params.PROVISIONING {
			return false, nil
		}
		if infraState == ves_io_schema_tf_params.ERRORED || infraState == ves_io_schema_tf_params.TIMED_OUT {
			return false, fmt.Errorf("evaluateTfParamsPlanStatus: terraform infra state failed: %s", ves_io_schema_tf_params.InfraState_name[int32(infraState)])
		}
		return true, nil
	default:
		return false, nil
	}
}

func evaluateTfParamsApplyStatus(action string, applyStatus *ves_io_schema_tf_params.ApplyStatus) (bool, string, error) {
	if applyStatus == nil {
		return false, "", nil
	}

	switch st := applyStatus.GetState().(type) {
	case *ves_io_schema_tf_params.ApplyStatus_ApplyState:
		applyState := st.ApplyState
		if applyState == ves_io_schema_tf_params.APPLY_PLANNING ||
			applyState == ves_io_schema_tf_params.APPLYING ||
			applyState == ves_io_schema_tf_params.APPLY_QUEUED {
			return false, "", nil
		}
		if applyState == ves_io_schema_tf_params.APPLY_ERRORED ||
			applyState == ves_io_schema_tf_params.APPLY_INIT_ERRORED ||
			applyState == ves_io_schema_tf_params.APPLY_PLAN_ERRORED {
			return false, "", fmt.Errorf("evaluateTfParamsApplyStatus: terraform apply failed: %s", applyStatus.GetErrorOutput())
		}
		if action == "apply" {
			return true, applyStatus.GetTfOutput(), nil
		}
		return true, "", nil
	case *ves_io_schema_tf_params.ApplyStatus_DestroyState:
		destroyState := st.DestroyState
		if destroyState == ves_io_schema_tf_params.DESTROYING || destroyState == ves_io_schema_tf_params.DESTROY_QUEUED {
			return false, "", nil
		}
		if destroyState == ves_io_schema_tf_params.DESTROY_ERRORED {
			return false, "", fmt.Errorf("evaluateTfParamsApplyStatus: terraform destroy failed: %s", applyStatus.GetErrorOutput())
		}
		return true, "", nil
	case *ves_io_schema_tf_params.ApplyStatus_InfraState:
		infraState := st.InfraState
		if infraState == ves_io_schema_tf_params.PROVISIONING {
			return false, "", nil
		}
		if infraState == ves_io_schema_tf_params.ERRORED || infraState == ves_io_schema_tf_params.TIMED_OUT {
			return false, "", fmt.Errorf("evaluateTfParamsApplyStatus: terraform infra state failed: %s", ves_io_schema_tf_params.InfraState_name[int32(infraState)])
		}
		return true, "", nil
	default:
		return false, "", nil
	}
}

func getTfParamsTerraformStatus(
	ctx context.Context,
	apiClient CustomAPIClient,
	namespace string,
	viewKind string,
	viewName string,
) (*ves_io_schema_tf_params.GetStatusResponse, error) {
	getReq := &ves_io_schema_tf_params.GetRequest{
		Namespace: normalizeTfParamsNamespace(namespace),
		ViewKind:  viewKind,
		ViewName:  viewName,
	}

	yamlGetReq, err := codec.ToYAML(getReq)
	if err != nil {
		return nil, fmt.Errorf("getTfParamsTerraformStatus: failed to marshal GetRequest: %w", err)
	}

	statusRespMsg, err := apiClient.CustomAPI(
		ctx,
		http.MethodGet,
		fmt.Sprintf(tfParamsGetStatusURI, getReq.GetNamespace(), getReq.GetViewKind(), getReq.GetViewName()),
		tfParamsGetStatusRPCFQN,
		yamlGetReq,
	)
	if err != nil {
		return nil, fmt.Errorf("getTfParamsTerraformStatus: custom API request failed: %w", err)
	}

	statusResp, ok := statusRespMsg.(*ves_io_schema_tf_params.GetStatusResponse)
	if !ok {
		return nil, fmt.Errorf("getTfParamsTerraformStatus: unexpected response type %T", statusRespMsg)
	}

	return statusResp, nil
}
