//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"fmt"
	"testing"

	"gopkg.volterra.us/stdlib/client/vesapi"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type virtualSiteDeleteStub struct {
	deleteErr error
	optsCount int
}

func (s *virtualSiteDeleteStub) DeleteObject(_ context.Context, _ string, _, _ string, opts ...vesapi.CallOpt) error {
	s.optsCount = len(opts)
	return s.deleteErr
}

// TestVirtualSiteOverrideDeleteHook_Success verifies a clean delete returns nil
// and passes vesapi.WithFailIfReferred() when failIfReferred is true.
func TestVirtualSiteOverrideDeleteHook_Success(t *testing.T) {
	stub := &virtualSiteDeleteStub{}
	err := VirtualSiteOverrideDeleteHook(context.Background(), stub, "default", "my-site", true)
	require.NoError(t, err)
	assert.Equal(t, 1, stub.optsCount)
}

// TestVirtualSiteOverrideDeleteHook_FailIfReferredFalse verifies that when
// failIfReferred is false, DeleteObject is called without vesapi.WithFailIfReferred().
func TestVirtualSiteOverrideDeleteHook_FailIfReferredFalse(t *testing.T) {
	stub := &virtualSiteDeleteStub{}
	err := VirtualSiteOverrideDeleteHook(context.Background(), stub, "default", "my-site", false)
	require.NoError(t, err)
	assert.Equal(t, 0, stub.optsCount)
}

// TestVirtualSiteOverrideDeleteHook_NotFound verifies that a "NotFound" error is silenced.
func TestVirtualSiteOverrideDeleteHook_NotFound(t *testing.T) {
	stub := &virtualSiteDeleteStub{deleteErr: fmt.Errorf("rpc error: code = NotFound desc = object not found")}
	err := VirtualSiteOverrideDeleteHook(context.Background(), stub, "default", "my-site", true)
	require.NoError(t, err)
}

// TestVirtualSiteOverrideDeleteHook_StatusCode404_Surfaced verifies that a bare
// "status code 404" error is NOT silenced (the API returns this for non-gone conditions).
func TestVirtualSiteOverrideDeleteHook_StatusCode404_Surfaced(t *testing.T) {
	stub := &virtualSiteDeleteStub{deleteErr: fmt.Errorf("status code 404")}
	err := VirtualSiteOverrideDeleteHook(context.Background(), stub, "default", "my-site", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status code 404")
}

// TestVirtualSiteOverrideDeleteHook_OtherError verifies that unrelated errors are surfaced.
func TestVirtualSiteOverrideDeleteHook_OtherError(t *testing.T) {
	stub := &virtualSiteDeleteStub{deleteErr: fmt.Errorf("status code 500: internal server error")}
	err := VirtualSiteOverrideDeleteHook(context.Background(), stub, "default", "my-site", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}
