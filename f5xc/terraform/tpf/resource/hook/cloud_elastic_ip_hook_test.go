//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"gopkg.volterra.us/stdlib/client/vesapi"

	ves_io_schema_cloud_elastic_ip "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/cloud_elastic_ip"
)

type cloudElasticIpDeleteStub struct {
	deleteCalls int
	getCalls    int
	customCalls []customAPICall

	deleteErr error
	getErrs   []error
	getResps  []vesapi.GetObjectResponse

	customResp proto.Message
	customErr  error
}

func (s *cloudElasticIpDeleteStub) DeleteObject(ctx context.Context, objType string, namespace, name string, opts ...vesapi.CallOpt) error {
	s.deleteCalls++
	return s.deleteErr
}

func (s *cloudElasticIpDeleteStub) GetObject(ctx context.Context, objType, namespace, name string, opts ...vesapi.CallOpt) (vesapi.GetObjectResponse, error) {
	s.getCalls++

	if len(s.getResps) > 0 {
		resp := s.getResps[0]
		s.getResps = s.getResps[1:]
		var err error
		if len(s.getErrs) > 0 {
			err = s.getErrs[0]
			s.getErrs = s.getErrs[1:]
		}
		return resp, err
	}

	if len(s.getErrs) > 0 {
		err := s.getErrs[0]
		s.getErrs = s.getErrs[1:]
		return nil, err
	}

	return &ves_io_schema_cloud_elastic_ip.DBObject{}, nil
}

func (s *cloudElasticIpDeleteStub) CustomAPI(ctx context.Context, method, uri, rpcFQN, reqYml string, opts ...vesapi.CallOpt) (proto.Message, error) {
	s.customCalls = append(s.customCalls, customAPICall{method: method, uri: uri, rpcFQN: rpcFQN, reqYml: reqYml})
	return s.customResp, s.customErr
}

func TestCloudElasticIpOverrideDeleteHook_Delete404IsSuccess(t *testing.T) {
	stub := &cloudElasticIpDeleteStub{deleteErr: fmt.Errorf("status code 404")}
	waitCalled := false
	originalWait := waitForCloudElasticIpPendingFinalizers
	waitForCloudElasticIpPendingFinalizers = func(
		ctx context.Context,
		name string,
		namespace string,
		uid string,
		getObject func(ctx context.Context, namespace, name string) (*ves_io_schema_cloud_elastic_ip.DBObject, error),
	) error {
		waitCalled = true
		return nil
	}
	defer func() { waitForCloudElasticIpPendingFinalizers = originalWait }()

	err := CloudElasticIpOverrideDeleteHook(context.Background(), stub, "ns1", "eip1", "uid1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if waitCalled {
		t.Fatalf("expected wait not to be called when delete returns 404")
	}
	if len(stub.customCalls) != 0 {
		t.Fatalf("expected no CustomAPI calls, got %d", len(stub.customCalls))
	}
}

func TestCloudElasticIpOverrideDeleteHook_FinalizersCompleteWithoutForceDelete(t *testing.T) {
	stub := &cloudElasticIpDeleteStub{}
	originalWait := waitForCloudElasticIpPendingFinalizers
	waitForCloudElasticIpPendingFinalizers = func(
		ctx context.Context,
		name string,
		namespace string,
		uid string,
		getObject func(ctx context.Context, namespace, name string) (*ves_io_schema_cloud_elastic_ip.DBObject, error),
	) error {
		return nil
	}
	defer func() { waitForCloudElasticIpPendingFinalizers = originalWait }()

	err := CloudElasticIpOverrideDeleteHook(context.Background(), stub, "ns1", "eip1", "uid1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(stub.customCalls) != 0 {
		t.Fatalf("expected no CustomAPI calls, got %d", len(stub.customCalls))
	}
	if stub.getCalls != 0 {
		t.Fatalf("expected no direct GetObject calls after successful finalizer wait, got %d", stub.getCalls)
	}
}

func TestCloudElasticIpOverrideDeleteHook_FinalizerTimeoutUsesForceDelete(t *testing.T) {
	stub := &cloudElasticIpDeleteStub{getErrs: []error{fmt.Errorf("status code 404")}}
	originalWait := waitForCloudElasticIpPendingFinalizers
	waitForCloudElasticIpPendingFinalizers = func(
		ctx context.Context,
		name string,
		namespace string,
		uid string,
		getObject func(ctx context.Context, namespace, name string) (*ves_io_schema_cloud_elastic_ip.DBObject, error),
	) error {
		return fmt.Errorf("resource %s still exists after %d retries, finalizers may not have completed", name, 30)
	}
	defer func() { waitForCloudElasticIpPendingFinalizers = originalWait }()

	err := CloudElasticIpOverrideDeleteHook(context.Background(), stub, "ns1", "eip1", "uid1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(stub.customCalls) != 1 {
		t.Fatalf("expected 1 CustomAPI call, got %d", len(stub.customCalls))
	}
	if stub.customCalls[0].rpcFQN != cloudElasticIpForceDeleteRPCFQN {
		t.Fatalf("expected rpc %q, got %q", cloudElasticIpForceDeleteRPCFQN, stub.customCalls[0].rpcFQN)
	}
	if stub.customCalls[0].uri != "/public/namespaces/system/cloud_elastic_ip/eip1/force-delete" {
		t.Fatalf("unexpected force-delete uri %q", stub.customCalls[0].uri)
	}
	if stub.getCalls != 1 {
		t.Fatalf("expected one post-force-delete GetObject call, got %d", stub.getCalls)
	}
}

func TestCloudElasticIpOverrideDeleteHook_PostForceDeleteCheckErrorIsReturned(t *testing.T) {
	stub := &cloudElasticIpDeleteStub{getErrs: []error{fmt.Errorf("status code 500")}}
	originalWait := waitForCloudElasticIpPendingFinalizers
	waitForCloudElasticIpPendingFinalizers = func(
		ctx context.Context,
		name string,
		namespace string,
		uid string,
		getObject func(ctx context.Context, namespace, name string) (*ves_io_schema_cloud_elastic_ip.DBObject, error),
	) error {
		return fmt.Errorf("resource %s still exists after %d retries, finalizers may not have completed", name, 30)
	}
	defer func() { waitForCloudElasticIpPendingFinalizers = originalWait }()

	err := CloudElasticIpOverrideDeleteHook(context.Background(), stub, "ns1", "eip1", "uid1")
	if err == nil || err.Error() != "CloudElasticIpOverrideDeleteHook: post-force-delete check failed: status code 500" {
		t.Fatalf("expected post-force-delete check error, got %v", err)
	}
	if len(stub.customCalls) != 1 {
		t.Fatalf("expected 1 CustomAPI call, got %d", len(stub.customCalls))
	}
}

func TestCloudElasticIpOverrideDeleteHook_FinalizerWaitErrorSkipsForceDelete(t *testing.T) {
	stub := &cloudElasticIpDeleteStub{}
	originalWait := waitForCloudElasticIpPendingFinalizers
	waitForCloudElasticIpPendingFinalizers = func(
		ctx context.Context,
		name string,
		namespace string,
		uid string,
		getObject func(ctx context.Context, namespace, name string) (*ves_io_schema_cloud_elastic_ip.DBObject, error),
	) error {
		return fmt.Errorf("error polling resource %q for finalizer completion: status code 500", uid)
	}
	defer func() { waitForCloudElasticIpPendingFinalizers = originalWait }()

	err := CloudElasticIpOverrideDeleteHook(context.Background(), stub, "ns1", "eip1", "uid1")
	if err == nil || err.Error() != "CloudElasticIpOverrideDeleteHook: waiting for pending finalizers failed: error polling resource \"uid1\" for finalizer completion: status code 500" {
		t.Fatalf("expected finalizer wait error, got %v", err)
	}
	if len(stub.customCalls) != 0 {
		t.Fatalf("expected no CustomAPI calls, got %d", len(stub.customCalls))
	}
}

func TestCloudElasticIpOverrideDeleteHook_ForceDeleteErrorIsReturned(t *testing.T) {
	stub := &cloudElasticIpDeleteStub{customErr: fmt.Errorf("status code 500")}
	originalWait := waitForCloudElasticIpPendingFinalizers
	waitForCloudElasticIpPendingFinalizers = func(
		ctx context.Context,
		name string,
		namespace string,
		uid string,
		getObject func(ctx context.Context, namespace, name string) (*ves_io_schema_cloud_elastic_ip.DBObject, error),
	) error {
		return fmt.Errorf("resource %s still exists after %d retries, finalizers may not have completed", name, 30)
	}
	defer func() { waitForCloudElasticIpPendingFinalizers = originalWait }()

	err := CloudElasticIpOverrideDeleteHook(context.Background(), stub, "ns1", "eip1", "uid1")
	if err == nil || err.Error() != "CloudElasticIpOverrideDeleteHook: force delete failed: status code 500" {
		t.Fatalf("expected force delete error, got %v", err)
	}
	if stub.getCalls != 0 {
		t.Fatalf("expected no post-force-delete GetObject call when force delete fails, got %d", stub.getCalls)
	}
}
