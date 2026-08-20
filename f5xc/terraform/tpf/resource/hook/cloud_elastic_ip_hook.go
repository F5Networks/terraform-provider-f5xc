//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"fmt"
	"strings"

	"gopkg.volterra.us/stdlib/client/vesapi"
	"gopkg.volterra.us/stdlib/codec"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/utils"
	ves_io_schema_cloud_elastic_ip "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/cloud_elastic_ip"
)

const (
	cloudElasticIpForceDeleteRPCFQN      = "ves.io.schema.cloud_elastic_ip.CustomAPI.ForceDeleteCloudElasticIP"
	cloudElasticIpForceDeleteURI         = "/public/namespaces/system/cloud_elastic_ip/%s/force-delete"
	cloudElasticIpFinalizerTimeoutMarker = "still exists after"
)

type cloudElasticIpDeleteClient interface {
	CustomAPIClient
	DeleteObject(ctx context.Context, objType string, namespace, name string, opts ...vesapi.CallOpt) error
	GetObject(ctx context.Context, objType, namespace, name string, opts ...vesapi.CallOpt) (vesapi.GetObjectResponse, error)
}

var waitForCloudElasticIpPendingFinalizers = func(
	ctx context.Context,
	name string,
	namespace string,
	uid string,
	getObject func(ctx context.Context, namespace, name string) (*ves_io_schema_cloud_elastic_ip.DBObject, error),
) error {
	return utils.WaitForPendingFinalizers(ctx, name, namespace, uid, getObject)
}

// CloudElasticIpOverrideDeleteHook executes the cloud_elastic_ip delete flow.
//
// Flow:
//  1. Run the standard DeleteObject request.
//  2. Wait for pending finalizers to complete.
//  3. If finalizers never clear within the normal wait window, call the force-delete
//     custom API and do a single follow-up GetObject check.
func CloudElasticIpOverrideDeleteHook(
	ctx context.Context,
	apiClient cloudElasticIpDeleteClient,
	namespace string,
	name string,
	uid string,
) error {
	deleteOpts := []vesapi.CallOpt{vesapi.WithFailIfReferred()}

	err := apiClient.DeleteObject(ctx, ves_io_schema_cloud_elastic_ip.ObjectType, namespace, name, deleteOpts...)
	if utils.IsNotFoundError(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("CloudElasticIpOverrideDeleteHook: standard delete failed: %w", err)
	}

	err = waitForCloudElasticIpPendingFinalizers(
		ctx,
		name,
		namespace,
		uid,
		func(ctx context.Context, namespace, name string) (*ves_io_schema_cloud_elastic_ip.DBObject, error) {
			resp, getErr := apiClient.GetObject(ctx, ves_io_schema_cloud_elastic_ip.ObjectType, namespace, name, deleteOpts...)
			if getErr != nil {
				return nil, getErr
			}

			dbObj, ok := resp.(*ves_io_schema_cloud_elastic_ip.DBObject)
			if !ok {
				return nil, fmt.Errorf("unexpected response type: %T", resp)
			}

			return dbObj, nil
		},
	)
	if err == nil {
		return nil
	}
	if !isCloudElasticIpFinalizerTimeout(err) {
		return fmt.Errorf("CloudElasticIpOverrideDeleteHook: waiting for pending finalizers failed: %w", err)
	}

	forceDeleteReq := &ves_io_schema_cloud_elastic_ip.ForceDeleteCloudElasticIPRequest{Name: name}
	yamlReq, err := codec.ToYAML(forceDeleteReq)
	if err != nil {
		return fmt.Errorf("CloudElasticIpOverrideDeleteHook: failed to marshal force-delete request: %w", err)
	}

	_, err = apiClient.CustomAPI(
		ctx,
		"POST",
		fmt.Sprintf(cloudElasticIpForceDeleteURI, name),
		cloudElasticIpForceDeleteRPCFQN,
		yamlReq,
	)
	if utils.IsNotFoundError(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("CloudElasticIpOverrideDeleteHook: force delete failed: %w", err)
	}

	_, err = apiClient.GetObject(ctx, ves_io_schema_cloud_elastic_ip.ObjectType, namespace, name, deleteOpts...)
	if utils.IsNotFoundError(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("CloudElasticIpOverrideDeleteHook: post-force-delete check failed: %w", err)
	}

	return nil
}

func isCloudElasticIpFinalizerTimeout(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(strings.ToLower(err.Error()), cloudElasticIpFinalizerTimeoutMarker)
}
