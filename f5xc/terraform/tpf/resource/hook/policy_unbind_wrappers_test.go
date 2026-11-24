//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
)

// This file holds the shared fake server and the shared checks for the exported
// *UnbindReferencingObjects hooks. Each hook test file defines its unbindWrapper
// and calls runUnbindWrapperSuite.

// unbindWrapper describes one exported *UnbindReferencingObjects function and the
// spec shape it edits on a referring parent object.
type unbindWrapper struct {
	call            func(ctx context.Context, c *client.APIClient, ns, name, errText string) error
	logPrefix       string
	parentSchema    string // ves.io.schema.<parentSchema>.Object in the 409 text
	parentKind      string // plural REST segment for parentSchema
	selfSchema      string // the policy's own schema type
	selfKind        string // plural REST segment for selfSchema
	selfIsFiltered  bool   // true when the wrapper skips its own kind
	containerKey    string
	listKey         string
	emptyChoiceKeys []string
}

const (
	testPolicyNS   = "default"
	testPolicyName = "p1"
	testParentNS   = "default"
	testParentName = "parent1"
)

// conflictText builds an F5XC 409 error text that lists refs as referrers.
func (w unbindWrapper) conflictText(refs ...string) string {
	text := fmt.Sprintf("DeleteResource: ves.io.schema.%s.Object tenant/%s/%s is still being referred by %d objects - [",
		w.selfSchema, testPolicyNS, testPolicyName, len(refs))
	for i, ref := range refs {
		if i > 0 {
			text += " "
		}
		text += ref
	}
	return text + "]"
}

// schemaRef returns a referrer entry for schema in namespace ns with the given name.
func schemaRef(schema, ns, name string) string {
	return fmt.Sprintf("ves.io.schema.%s.Object %s/%s", schema, ns, name)
}

func (w unbindWrapper) parentRef() string {
	return schemaRef(w.parentSchema, testParentNS, testParentName)
}

func (w unbindWrapper) parentPath(prefix string) string {
	return objectPath(prefix, testParentNS, w.parentKind, testParentName)
}

// objectPath returns the REST path of an object under the /config or /public prefix.
func objectPath(prefix, ns, kind, name string) string {
	return fmt.Sprintf("/%s/namespaces/%s/%s/%s", prefix, ns, kind, name)
}

// parentWithPolicies returns a parent object whose spec references the given policy names.
func (w unbindWrapper) parentWithPolicies(names ...string) map[string]any {
	list := make([]any, 0, len(names))
	for _, n := range names {
		list = append(list, policyObjRef(testPolicyNS, n))
	}
	return map[string]any{
		"metadata": map[string]any{"name": testParentName, "namespace": testParentNS},
		"spec": map[string]any{
			w.containerKey: map[string]any{w.listKey: list},
		},
	}
}

// fakeResponse is a canned reply for one method + path.
type fakeResponse struct {
	status int
	body   string
}

// fakeConfigServer serves canned GET/PUT replies and records every request.
// Unregistered routes return 404.
type fakeConfigServer struct {
	mu       sync.Mutex
	routes   map[string]fakeResponse // key: "METHOD path"
	requests []string
	putBody  map[string][]byte
}

func newFakeConfigServer() *fakeConfigServer {
	return &fakeConfigServer{routes: map[string]fakeResponse{}, putBody: map[string][]byte{}}
}

func (f *fakeConfigServer) on(method, path string, status int, body string) {
	f.routes[method+" "+path] = fakeResponse{status: status, body: body}
}

func (f *fakeConfigServer) record(method, path string, body []byte) (fakeResponse, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := method + " " + path
	f.requests = append(f.requests, key)
	if method == http.MethodPut {
		f.putBody[path] = body
	}
	resp, ok := f.routes[key]
	return resp, ok
}

func (f *fakeConfigServer) snapshot() ([]string, map[string][]byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	reqs := append([]string(nil), f.requests...)
	bodies := make(map[string][]byte, len(f.putBody))
	for k, v := range f.putBody {
		bodies[k] = v
	}
	return reqs, bodies
}

func (f *fakeConfigServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	resp, ok := f.record(r.Method, r.URL.Path, body)
	if !ok {
		resp = fakeResponse{status: http.StatusNotFound, body: `{"error":"not found"}`}
	}
	w.WriteHeader(resp.status)
	_, _ = w.Write([]byte(resp.body))
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

// startFakeConfigServer starts f and returns an APIClient that targets it.
func startFakeConfigServer(t *testing.T, f *fakeConfigServer) *client.APIClient {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return newUnbindTestAPIClient(srv)
}

// decodePutSpec returns the spec map of the PUT body sent to path.
func decodePutSpec(t *testing.T, bodies map[string][]byte, path string) map[string]any {
	t.Helper()
	raw, ok := bodies[path]
	require.True(t, ok, "expected PUT to %s", path)
	var parent map[string]any
	require.NoError(t, json.Unmarshal(raw, &parent))
	spec, ok := parent["spec"].(map[string]any)
	require.True(t, ok, "PUT body must keep spec")
	return spec
}

// runUnbindWrapperSuite runs the checks that every *UnbindReferencingObjects hook
// must pass.
func runUnbindWrapperSuite(t *testing.T, w unbindWrapper) {
	t.Run("nil client", func(t *testing.T) { checkUnbindNilClient(t, w) })
	t.Run("no referrers", func(t *testing.T) { checkUnbindNoReferrers(t, w) })
	t.Run("last policy switches to empty choice", func(t *testing.T) { checkUnbindLastPolicy(t, w) })
	t.Run("keeps other policies", func(t *testing.T) { checkUnbindKeepsOtherPolicies(t, w) })
	t.Run("public fallback", func(t *testing.T) { checkUnbindPublicFallback(t, w) })
	t.Run("skipped referrers", func(t *testing.T) { checkUnbindSkippedReferrers(t, w) })
	t.Run("put error", func(t *testing.T) { checkUnbindPutError(t, w) })
	t.Run("partial failure", func(t *testing.T) { checkUnbindPartialFailure(t, w) })
	t.Run("self kind filter", func(t *testing.T) { checkUnbindSelfKindFilter(t, w) })
	t.Run("referrer namespace", func(t *testing.T) { checkUnbindReferrerNamespace(t, w) })
}

// checkUnbindNilClient verifies that a nil client or nil RestClient returns an
// error that names the policy type.
func checkUnbindNilClient(t *testing.T, w unbindWrapper) {
	for _, c := range []*client.APIClient{nil, {}} {
		err := w.call(context.Background(), c, testPolicyNS, testPolicyName, w.conflictText(w.parentRef()))
		require.Error(t, err)
		assert.Contains(t, err.Error(), w.logPrefix+"UnbindReferencingObjects")
	}
}

// checkUnbindNoReferrers verifies that a conflict with no parseable referrers is
// a no-op and sends no HTTP requests.
func checkUnbindNoReferrers(t *testing.T, w unbindWrapper) {
	f := newFakeConfigServer()
	c := startFakeConfigServer(t, f)

	err := w.call(context.Background(), c, testPolicyNS, testPolicyName, "conflict: object is in use")

	require.NoError(t, err)
	reqs, _ := f.snapshot()
	assert.Empty(t, reqs)
}

// checkUnbindLastPolicy verifies that removing the only policy deletes the
// container and sets every empty-choice key, using /config for GET and PUT.
func checkUnbindLastPolicy(t *testing.T, w unbindWrapper) {
	f := newFakeConfigServer()
	f.on(http.MethodGet, w.parentPath("config"), http.StatusOK, mustJSON(t, w.parentWithPolicies(testPolicyName)))
	f.on(http.MethodPut, w.parentPath("config"), http.StatusOK, `{}`)
	c := startFakeConfigServer(t, f)

	err := w.call(context.Background(), c, testPolicyNS, testPolicyName, w.conflictText(w.parentRef()))

	require.NoError(t, err)
	reqs, bodies := f.snapshot()
	assert.Equal(t, []string{
		http.MethodGet + " " + w.parentPath("config"),
		http.MethodPut + " " + w.parentPath("config"),
	}, reqs)

	spec := decodePutSpec(t, bodies, w.parentPath("config"))
	assert.NotContains(t, spec, w.containerKey)
	for _, key := range w.emptyChoiceKeys {
		assert.Equal(t, map[string]any{}, spec[key], "expected empty choice %q", key)
	}
}

// checkUnbindKeepsOtherPolicies verifies that only the deleted policy is pruned
// and the container stays when other policies remain.
func checkUnbindKeepsOtherPolicies(t *testing.T, w unbindWrapper) {
	f := newFakeConfigServer()
	f.on(http.MethodGet, w.parentPath("config"), http.StatusOK, mustJSON(t, w.parentWithPolicies("other", testPolicyName)))
	f.on(http.MethodPut, w.parentPath("config"), http.StatusOK, `{}`)
	c := startFakeConfigServer(t, f)

	err := w.call(context.Background(), c, testPolicyNS, testPolicyName, w.conflictText(w.parentRef()))

	require.NoError(t, err)
	_, bodies := f.snapshot()
	spec := decodePutSpec(t, bodies, w.parentPath("config"))
	container, ok := spec[w.containerKey].(map[string]any)
	require.True(t, ok, "container %q must remain", w.containerKey)
	assert.Equal(t, []string{"other"}, policyNamesInList(t, container[w.listKey]))
	for _, key := range w.emptyChoiceKeys {
		assert.NotContains(t, spec, key)
	}
}

// checkUnbindPublicFallback verifies that a /config GET failure falls back to
// /public, and the PUT targets the URL that served the GET.
func checkUnbindPublicFallback(t *testing.T, w unbindWrapper) {
	f := newFakeConfigServer()
	f.on(http.MethodGet, w.parentPath("public"), http.StatusOK, mustJSON(t, w.parentWithPolicies(testPolicyName)))
	f.on(http.MethodPut, w.parentPath("public"), http.StatusOK, `{}`)
	c := startFakeConfigServer(t, f)

	err := w.call(context.Background(), c, testPolicyNS, testPolicyName, w.conflictText(w.parentRef()))

	require.NoError(t, err)
	reqs, bodies := f.snapshot()
	assert.Equal(t, []string{
		http.MethodGet + " " + w.parentPath("config"),
		http.MethodGet + " " + w.parentPath("public"),
		http.MethodPut + " " + w.parentPath("public"),
	}, reqs)
	assert.Contains(t, bodies, w.parentPath("public"))
}

// checkUnbindSkippedReferrers verifies that unreadable or irrelevant referrers
// are skipped without a PUT and without an error.
func checkUnbindSkippedReferrers(t *testing.T, w unbindWrapper) {
	tests := []struct {
		name    string
		getCode int
		getBody string
	}{
		{name: "get 403 on both endpoints", getCode: http.StatusForbidden, getBody: `{}`},
		{name: "get 500 on both endpoints", getCode: http.StatusInternalServerError, getBody: `{}`},
		{name: "invalid json", getCode: http.StatusOK, getBody: `{not json`},
		{name: "missing spec", getCode: http.StatusOK, getBody: `{"metadata":{}}`},
		{name: "policy not referenced", getCode: http.StatusOK, getBody: mustJSON(t, w.parentWithPolicies("other"))},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeConfigServer()
			f.on(http.MethodGet, w.parentPath("config"), tc.getCode, tc.getBody)
			f.on(http.MethodGet, w.parentPath("public"), tc.getCode, tc.getBody)
			c := startFakeConfigServer(t, f)

			err := w.call(context.Background(), c, testPolicyNS, testPolicyName, w.conflictText(w.parentRef()))

			require.NoError(t, err)
			_, bodies := f.snapshot()
			assert.Empty(t, bodies, "no PUT expected")
		})
	}
}

// checkUnbindPutError verifies that a failed PUT (4xx or 5xx) aborts the unbind
// and returns an error that names the parent object.
// checkUnbindPartialFailure verifies the behavior when one of several parents
// fails to save: parents before it stay saved, the error names the failed
// parent, and parents after it are not fetched. The unbind does not roll back.
func checkUnbindPartialFailure(t *testing.T, w unbindWrapper) {
	parents := []string{"p1", "p2", "p3"}
	pathOf := func(name string) string { return objectPath("config", testParentNS, w.parentKind, name) }

	f := newFakeConfigServer()
	for _, name := range parents {
		f.on(http.MethodGet, pathOf(name), http.StatusOK, mustJSON(t, w.parentWithPolicies(testPolicyName)))
	}
	f.on(http.MethodPut, pathOf("p1"), http.StatusOK, `{}`)
	f.on(http.MethodPut, pathOf("p2"), http.StatusInternalServerError, `{"error":"boom"}`)
	c := startFakeConfigServer(t, f)

	refs := make([]string, 0, len(parents))
	for _, name := range parents {
		refs = append(refs, schemaRef(w.parentSchema, testParentNS, name))
	}

	err := w.call(context.Background(), c, testPolicyNS, testPolicyName, w.conflictText(refs...))

	require.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("failed to update %s %s/p2", w.parentKind, testParentNS))

	reqs, bodies := f.snapshot()
	assert.Equal(t, []string{
		http.MethodGet + " " + pathOf("p1"),
		http.MethodPut + " " + pathOf("p1"),
		http.MethodGet + " " + pathOf("p2"),
		http.MethodPut + " " + pathOf("p2"),
	}, reqs, "p3 must not be fetched after p2 fails")

	spec := decodePutSpec(t, bodies, pathOf("p1"))
	assert.NotContains(t, spec, w.containerKey, "p1 keeps its saved change")
}

func checkUnbindPutError(t *testing.T, w unbindWrapper) {
	for _, status := range []int{http.StatusForbidden, http.StatusConflict, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			f := newFakeConfigServer()
			f.on(http.MethodGet, w.parentPath("config"), http.StatusOK, mustJSON(t, w.parentWithPolicies(testPolicyName)))
			f.on(http.MethodPut, w.parentPath("config"), status, `{"error":"rejected"}`)
			c := startFakeConfigServer(t, f)

			err := w.call(context.Background(), c, testPolicyNS, testPolicyName, w.conflictText(w.parentRef()))

			require.Error(t, err)
			assert.Contains(t, err.Error(), fmt.Sprintf("failed to update %s %s/%s", w.parentKind, testParentNS, testParentName))
		})
	}
}

// checkUnbindSelfKindFilter verifies that a filtered wrapper never fetches the
// policy's own kind, and an unfiltered wrapper fetches every parsed referrer.
func checkUnbindSelfKindFilter(t *testing.T, w unbindWrapper) {
	f := newFakeConfigServer()
	c := startFakeConfigServer(t, f)

	err := w.call(context.Background(), c, testPolicyNS, testPolicyName,
		w.conflictText(schemaRef(w.selfSchema, testPolicyNS, "sibling")))

	require.NoError(t, err)
	reqs, _ := f.snapshot()
	if w.selfIsFiltered {
		assert.Empty(t, reqs)
		return
	}
	assert.Contains(t, reqs, http.MethodGet+" "+objectPath("config", testPolicyNS, w.selfKind, "sibling"))
}

// checkUnbindReferrerNamespace verifies that the referrer's own namespace, not
// the policy namespace, builds the parent URL.
func checkUnbindReferrerNamespace(t *testing.T, w unbindWrapper) {
	f := newFakeConfigServer()
	c := startFakeConfigServer(t, f)

	err := w.call(context.Background(), c, testPolicyNS, testPolicyName,
		w.conflictText(schemaRef(w.parentSchema, "system", testParentName)))

	require.NoError(t, err)
	reqs, _ := f.snapshot()
	assert.Contains(t, reqs, http.MethodGet+" "+objectPath("config", "system", w.parentKind, testParentName))
}

// checkUnbindReferrerKinds verifies that every kind in accepted is fetched and
// every kind in rejected is skipped without an HTTP request.
func checkUnbindReferrerKinds(t *testing.T, w unbindWrapper, accepted, rejected map[string]string) {
	for kind, schema := range accepted {
		t.Run("accepts "+kind, func(t *testing.T) {
			f := newFakeConfigServer()
			c := startFakeConfigServer(t, f)

			err := w.call(context.Background(), c, testPolicyNS, testPolicyName, w.conflictText(schemaRef(schema, testParentNS, "obj")))

			require.NoError(t, err)
			reqs, _ := f.snapshot()
			assert.Contains(t, reqs, http.MethodGet+" "+objectPath("config", testParentNS, kind, "obj"))
		})
	}
	for kind, schema := range rejected {
		t.Run("rejects "+kind, func(t *testing.T) {
			f := newFakeConfigServer()
			c := startFakeConfigServer(t, f)

			err := w.call(context.Background(), c, testPolicyNS, testPolicyName, w.conflictText(schemaRef(schema, testParentNS, "obj")))

			require.NoError(t, err)
			reqs, _ := f.snapshot()
			assert.Empty(t, reqs)
		})
	}
}
