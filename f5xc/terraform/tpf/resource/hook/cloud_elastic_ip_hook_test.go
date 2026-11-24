//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.volterra.us/stdlib/client/vesapi"

	ves_io_schema_cloud_elastic_ip "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/cloud_elastic_ip"
	ves_io_schema_virtual_site "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/virtual_site"
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

// stubFinalizerWait replaces the finalizer wait for the test and restores it afterwards.
func stubFinalizerWait(t *testing.T, fn func(getObject func(context.Context, string, string) (*ves_io_schema_cloud_elastic_ip.DBObject, error)) error) {
	t.Helper()
	orig := waitForCloudElasticIpPendingFinalizers
	waitForCloudElasticIpPendingFinalizers = func(ctx context.Context, _, _, _ string,
		getObject func(context.Context, string, string) (*ves_io_schema_cloud_elastic_ip.DBObject, error)) error {
		return fn(getObject)
	}
	t.Cleanup(func() { waitForCloudElasticIpPendingFinalizers = orig })
}

// The finalizer getter returns the DBObject, passes through errors, and rejects other types.
func TestCloudElasticIpOverrideDeleteHook_FinalizerGetter(t *testing.T) {
	boom := errors.New("boom")
	stub := &cloudElasticIpDeleteStub{
		getResps: []vesapi.GetObjectResponse{&ves_io_schema_cloud_elastic_ip.DBObject{}, nil, &ves_io_schema_virtual_site.DBObject{}},
		getErrs:  []error{nil, boom, nil},
	}
	stubFinalizerWait(t, func(get func(context.Context, string, string) (*ves_io_schema_cloud_elastic_ip.DBObject, error)) error {
		obj, err := get(context.Background(), "ns1", "eip1")
		require.NoError(t, err)
		require.NotNil(t, obj)

		_, err = get(context.Background(), "ns1", "eip1")
		require.ErrorIs(t, err, boom)

		_, err = get(context.Background(), "ns1", "eip1")
		require.ErrorContains(t, err, "unexpected response type")
		return nil
	})

	require.NoError(t, CloudElasticIpOverrideDeleteHook(context.Background(), stub, "ns1", "eip1", "uid1"))
	assert.Equal(t, 3, stub.getCalls)
}

// A non-404 delete error is returned without a finalizer wait.
func TestCloudElasticIpOverrideDeleteHook_DeleteError(t *testing.T) {
	stub := &cloudElasticIpDeleteStub{deleteErr: errors.New("status code 500")}
	stubFinalizerWait(t, func(func(context.Context, string, string) (*ves_io_schema_cloud_elastic_ip.DBObject, error)) error {
		t.Fatal("finalizer wait must not run after a failed delete")
		return nil
	})
	require.ErrorContains(t, CloudElasticIpOverrideDeleteHook(context.Background(), stub, "ns1", "eip1", "uid1"), "standard delete failed")
}

// A 404 from force delete and from the follow-up GET both mean success.
func TestCloudElasticIpOverrideDeleteHook_ForceDeleteNotFound(t *testing.T) {
	timeout := func(func(context.Context, string, string) (*ves_io_schema_cloud_elastic_ip.DBObject, error)) error {
		return errors.New("resource eip1 still exists after 30 retries")
	}

	t.Run("force delete 404", func(t *testing.T) {
		stubFinalizerWait(t, timeout)
		stub := &cloudElasticIpDeleteStub{customErr: errors.New("status code 404")}
		require.NoError(t, CloudElasticIpOverrideDeleteHook(context.Background(), stub, "ns1", "eip1", "uid1"))
		assert.Zero(t, stub.getCalls)
	})
	t.Run("object still readable", func(t *testing.T) {
		stubFinalizerWait(t, timeout)
		stub := &cloudElasticIpDeleteStub{}
		require.NoError(t, CloudElasticIpOverrideDeleteHook(context.Background(), stub, "ns1", "eip1", "uid1"))
		assert.Equal(t, 1, stub.getCalls)
	})
}

// A nil error is not a finalizer timeout.
func TestIsCloudElasticIpFinalizerTimeout_Nil(t *testing.T) {
	assert.False(t, isCloudElasticIpFinalizerTimeout(nil))
}

// The default finalizer wait returns nil once the object is gone (404).
func TestWaitForCloudElasticIpPendingFinalizers_Default(t *testing.T) {
	if testing.Short() {
		t.Skip("the production wait sleeps for its initial backoff")
	}
	err := waitForCloudElasticIpPendingFinalizers(context.Background(), "eip1", "ns1", "uid1",
		func(context.Context, string, string) (*ves_io_schema_cloud_elastic_ip.DBObject, error) {
			return nil, errors.New("status code 404")
		})
	require.NoError(t, err)
}
