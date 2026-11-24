//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package client

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.volterra.us/stdlib/client/vesapi"
	"gopkg.volterra.us/stdlib/db"
	"gopkg.volterra.us/stdlib/server"
	"gopkg.volterra.us/stdlib/svcfw"

	ves_io_schema_api_credential "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/api_credential"
	ves_io_schema_dns_zone_rrset "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/dns_zone/rrset"
	ves_io_schema_known_label "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/known_label"
	ves_io_schema_known_label_key "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/known_label_key"
	ves_io_schema_aws_tgw_site "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/views/aws_tgw_site"
)

// TestStubServers_Panic verifies that the unimplemented stub RPCs panic, so a
// test that reaches them fails loudly instead of using an empty response.
func TestStubServers_Panic(t *testing.T) {
	ctx := context.Background()
	s := &apiCredentialCustomAPIServer{}
	ns := &namespaceCustomAPIServer{}

	calls := map[string]func(){
		"CreateKubeConfig":                     func() { _, _ = s.CreateKubeConfig(ctx, nil) },
		"CreateKubeConfigServiceCredentials":   func() { _, _ = s.CreateKubeConfigServiceCredentials(ctx, nil) },
		"CreateSiteGlobalKubeConfig":           func() { _, _ = s.CreateSiteGlobalKubeConfig(ctx, nil) },
		"GetKubeConfig":                        func() { _, _ = s.GetKubeConfig(ctx, nil) },
		"GetKubeConfigServiceCredentials":      func() { _, _ = s.GetKubeConfigServiceCredentials(ctx, nil) },
		"ListKubeConfig":                       func() { _, _ = s.ListKubeConfig(ctx, nil) },
		"ListKubeConfigServiceCredentials":     func() { _, _ = s.ListKubeConfigServiceCredentials(ctx, nil) },
		"RevokeKubeConfig":                     func() { _, _ = s.RevokeKubeConfig(ctx, nil) },
		"RevokeKubeConfigServiceCredentials":   func() { _, _ = s.RevokeKubeConfigServiceCredentials(ctx, nil) },
		"ActivateKubeConfig":                   func() { _, _ = s.ActivateKubeConfig(ctx, nil) },
		"ActivateKubeConfigServiceCredentials": func() { _, _ = s.ActivateKubeConfigServiceCredentials(ctx, nil) },
		"GetSiteGlobalKubeConfig":              func() { _, _ = s.GetSiteGlobalKubeConfig(ctx, nil) },
		"ListSiteGlobalKubeConfig":             func() { _, _ = s.ListSiteGlobalKubeConfig(ctx, nil) },
		"RevokeSiteGlobalKubeConfig":           func() { _, _ = s.RevokeSiteGlobalKubeConfig(ctx, nil) },
		"ActivateSiteGlobalKubeConfig":         func() { _, _ = s.ActivateSiteGlobalKubeConfig(ctx, nil) },
		"DynamicData":                          func() { _, _ = ns.DynamicData(ctx, nil) },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			assert.Panics(t, call)
		})
	}
}

// TestStubServers_ReturnEmptyResponses verifies that every implemented stub
// RPC returns a non-nil response and no error.
func TestStubServers_ReturnEmptyResponses(t *testing.T) {
	ctx := context.Background()
	ac := &apiCredentialCustomAPIServer{}
	vh := &vhCustomAPIServer{}
	site := &siteCustomAPIServer{}
	nsAPI := &nsCustomAPIServer{}
	vpc := &vpcCustomAPIServer{}
	vnet := &vnetCustomAPIServer{}
	tgw := &tgwCustomAPIServer{}
	tf := &tfCustomAPIServer{}
	tfAction := &tfCustomActionAPIServer{}
	ns := &namespaceCustomAPIServer{}
	gcp := &gcpVpcCustomAPIServer{}
	dns := &dnsZoneRecordCustomAPIServer{}
	kl := &knownLabelCustomAPIServer{}
	klk := &knownLabelKeyCustomAPIServer{}

	calls := map[string]func() (any, error){
		"ac.Create":                       func() (any, error) { return ac.Create(ctx, nil) },
		"ac.List":                         func() (any, error) { return ac.List(ctx, nil) },
		"ac.Revoke":                       func() (any, error) { return ac.Revoke(ctx, nil) },
		"ac.Activate":                     func() (any, error) { return ac.Activate(ctx, nil) },
		"ac.Renew":                        func() (any, error) { return ac.Renew(ctx, nil) },
		"ac.CreateServiceCredentials":     func() (any, error) { return ac.CreateServiceCredentials(ctx, nil) },
		"ac.ListServiceCredentials":       func() (any, error) { return ac.ListServiceCredentials(ctx, nil) },
		"ac.ActivateServiceCredentials":   func() (any, error) { return ac.ActivateServiceCredentials(ctx, nil) },
		"ac.RenewServiceCredentials":      func() (any, error) { return ac.RenewServiceCredentials(ctx, nil) },
		"ac.RevokeServiceCredentials":     func() (any, error) { return ac.RevokeServiceCredentials(ctx, nil) },
		"ac.RecreateScimToken":            func() (any, error) { return ac.RecreateScimToken(ctx, nil) },
		"ac.RevokeScimToken":              func() (any, error) { return ac.RevokeScimToken(ctx, nil) },
		"ac.GetScimToken":                 func() (any, error) { return ac.GetScimToken(ctx, nil) },
		"ac.GetServiceCredentials":        func() (any, error) { return ac.GetServiceCredentials(ctx, nil) },
		"ac.ReplaceServiceCredentials":    func() (any, error) { return ac.ReplaceServiceCredentials(ctx, nil) },
		"ac.BulkRevoke":                   func() (any, error) { return ac.BulkRevoke(ctx, nil) },
		"ac.BulkRevokeServiceCredentials": func() (any, error) { return ac.BulkRevokeServiceCredentials(ctx, nil) },
		"vh.GetDnsInfo":                   func() (any, error) { return vh.GetDnsInfo(ctx, nil) },
		"vh.AssignAPIDefinition":          func() (any, error) { return vh.AssignAPIDefinition(ctx, nil) },
		"site.SetState":                   func() (any, error) { return site.SetState(ctx, nil) },
		"nsAPI.CascadeDelete":             func() (any, error) { return nsAPI.CascadeDelete(ctx, nil) },
		"nsAPI.EvaluateAPIAccess":         func() (any, error) { return nsAPI.EvaluateAPIAccess(ctx, nil) },
		"nsAPI.EvaluateBatchAPIAccess":    func() (any, error) { return nsAPI.EvaluateBatchAPIAccess(ctx, nil) },
		"nsAPI.SuggestValues":             func() (any, error) { return nsAPI.SuggestValues(ctx, nil) },
		"nsAPI.LookupUserRoles":           func() (any, error) { return nsAPI.LookupUserRoles(ctx, nil) },
		"vpc.SetVPCK8SHostnames":          func() (any, error) { return vpc.SetVPCK8SHostnames(ctx, nil) },
		"vpc.SetVIPInfo":                  func() (any, error) { return vpc.SetVIPInfo(ctx, nil) },
		"vpc.SetCloudSiteInfo":            func() (any, error) { return vpc.SetCloudSiteInfo(ctx, nil) },
		"vpc.ValidateConfig":              func() (any, error) { return vpc.ValidateConfig(ctx, nil) },
		"vnet.SetVIPInfo":                 func() (any, error) { return vnet.SetVIPInfo(ctx, nil) },
		"vnet.SetCloudSiteInfo":           func() (any, error) { return vnet.SetCloudSiteInfo(ctx, nil) },
		"vnet.ValidateConfig":             func() (any, error) { return vnet.ValidateConfig(ctx, nil) },
		"tgw.SetVPNTunnels":               func() (any, error) { return tgw.SetVPNTunnels(ctx, nil) },
		"tgw.SetTGWInfo":                  func() (any, error) { return tgw.SetTGWInfo(ctx, nil) },
		"tgw.SetVIPInfo":                  func() (any, error) { return tgw.SetVIPInfo(ctx, nil) },
		"tgw.ValidateConfig":              func() (any, error) { return tgw.ValidateConfig(ctx, nil) },
		"tf.Get":                          func() (any, error) { return tf.Get(ctx, nil) },
		"tf.GetStatus":                    func() (any, error) { return tf.GetStatus(ctx, nil) },
		"tfAction.Run":                    func() (any, error) { return tfAction.Run(ctx, nil) },
		"tfAction.ForceDelete":            func() (any, error) { return tfAction.ForceDelete(ctx, nil) },
		"ns.SetFastACLsForInternetVIPs":   func() (any, error) { return ns.SetFastACLsForInternetVIPs(ctx, nil) },
		"ns.GetFastACLsForInternetVIPs":   func() (any, error) { return ns.GetFastACLsForInternetVIPs(ctx, nil) },
		"ns.SetActiveServicePolicies":     func() (any, error) { return ns.SetActiveServicePolicies(ctx, nil) },
		"ns.GetActiveServicePolicies":     func() (any, error) { return ns.GetActiveServicePolicies(ctx, nil) },
		"ns.SetActiveNetworkPolicies":     func() (any, error) { return ns.SetActiveNetworkPolicies(ctx, nil) },
		"ns.GetActiveNetworkPolicies":     func() (any, error) { return ns.GetActiveNetworkPolicies(ctx, nil) },
		"ns.SetActiveAlertPolicies":       func() (any, error) { return ns.SetActiveAlertPolicies(ctx, nil) },
		"ns.GetActiveAlertPolicies":       func() (any, error) { return ns.GetActiveAlertPolicies(ctx, nil) },
		"ns.CascadeDelete":                func() (any, error) { return ns.CascadeDelete(ctx, nil) },
		"ns.SuggestValues":                func() (any, error) { return ns.SuggestValues(ctx, nil) },
		"ns.ValidateRules":                func() (any, error) { return ns.ValidateRules(ctx, nil) },
		"ns.UpdateAllowAdvertiseOnPublic": func() (any, error) { return ns.UpdateAllowAdvertiseOnPublic(ctx, nil) },
		"ns.NetworkingInventory":          func() (any, error) { return ns.NetworkingInventory(ctx, nil) },
		"ns.ApplicationInventory":         func() (any, error) { return ns.ApplicationInventory(ctx, nil) },
		"ns.AllApplicationInventory":      func() (any, error) { return ns.AllApplicationInventory(ctx, nil) },
		"ns.AllApplicationInventoryWaf":   func() (any, error) { return ns.AllApplicationInventoryWaf(ctx, nil) },
		"ns.OIDCOAuthDiscovery":           func() (any, error) { return ns.OIDCOAuthDiscovery(ctx, nil) },
		"ns.GetDynamicDefaultValues":      func() (any, error) { return ns.GetDynamicDefaultValues(ctx, nil) },
		"gcp.SetCloudSiteInfo":            func() (any, error) { return gcp.SetCloudSiteInfo(ctx, nil) },
		"gcp.ValidateConfig":              func() (any, error) { return gcp.ValidateConfig(ctx, nil) },
		"dns.Create":                      func() (any, error) { return dns.Create(ctx, nil) },
		"dns.Get":                         func() (any, error) { return dns.Get(ctx, nil) },
		"dns.Replace":                     func() (any, error) { return dns.Replace(ctx, nil) },
		"dns.Delete":                      func() (any, error) { return dns.Delete(ctx, nil) },
		"kl.Create":                       func() (any, error) { return kl.Create(ctx, nil) },
		"kl.Delete":                       func() (any, error) { return kl.Delete(ctx, nil) },
		"klk.Create":                      func() (any, error) { return klk.Create(ctx, nil) },
		"klk.Delete":                      func() (any, error) { return klk.Delete(ctx, nil) },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			rsp, err := call()
			require.NoError(t, err)
			assert.NotNil(t, rsp)
		})
	}
}

// TestAPICredentialStub_GetSetsExpiration verifies that the api_credential Get
// stub returns an expiration timestamp, which credential resources read.
func TestAPICredentialStub_GetSetsExpiration(t *testing.T) {
	rsp, err := (&apiCredentialCustomAPIServer{}).Get(context.Background(), &ves_io_schema_api_credential.GetRequest{})

	require.NoError(t, err)
	assert.NotNil(t, rsp.GetObject().GetSpec().GetGcSpec().GetExpirationTimestamp())
}

// TestKnownLabelStubs_EchoRequest verifies that the known_label and
// known_label_key Get stubs echo the requested key and value.
func TestKnownLabelStubs_EchoRequest(t *testing.T) {
	ctx := context.Background()

	lbl, err := (&knownLabelCustomAPIServer{}).Get(ctx, &ves_io_schema_known_label.GetRequest{Key: "k", Value: "v"})
	require.NoError(t, err)
	require.Len(t, lbl.Label, 1)
	assert.Equal(t, "k", lbl.Label[0].Key)
	assert.Equal(t, "v", lbl.Label[0].Value)

	key, err := (&knownLabelKeyCustomAPIServer{}).Get(ctx, &ves_io_schema_known_label_key.GetRequest{Key: "k"})
	require.NoError(t, err)
	require.Len(t, key.LabelKey, 1)
	assert.Equal(t, "k", key.LabelKey[0].Key)
}

// TestDNSZoneRecordStub_AcceptsRequests verifies that the rrset stub accepts
// typed requests, not only nil requests.
func TestDNSZoneRecordStub_AcceptsRequests(t *testing.T) {
	rsp, err := (&dnsZoneRecordCustomAPIServer{}).Get(context.Background(), &ves_io_schema_dns_zone_rrset.GetRequest{})

	require.NoError(t, err)
	assert.NotNil(t, rsp)
}

// fakeTGWService is a svcfw.Service that implements only the db calls used by
// tgwCustomAPIServer.SetVPCIpPrefixes. Any other call panics on the nil
// embedded interface.
type fakeTGWService struct {
	svcfw.Service

	entry   db.Entry
	found   bool
	findErr error
	chgErr  error
	changed db.Entry
}

func (f *fakeTGWService) FindEntry(_ context.Context, _, _ string, _ ...db.FindEntryOpt) (db.Entry, bool, error) {
	return f.entry, f.found, f.findErr
}

func (f *fakeTGWService) ChgEntry(_ context.Context, e db.Entry, _ ...db.ChgEntryOpt) (db.Entry, error) {
	f.changed = e
	return e, f.chgErr
}

// TestTGWStub_SetVPCIpPrefixes verifies that the stub requires a tenant, maps
// db lookup and update failures to errors, and stores the prefixes on success.
func TestTGWStub_SetVPCIpPrefixes(t *testing.T) {
	hdr := http.Header{}
	hdr.Set(server.TenantHdrName, "acme")
	tenantCtx := server.ContextFromRestHdrs(context.Background(), hdr)

	newEntry := func() db.Entry {
		obj := &ves_io_schema_aws_tgw_site.Object{
			Spec: &ves_io_schema_aws_tgw_site.SpecType{GcSpec: &ves_io_schema_aws_tgw_site.GlobalSpecType{}},
		}
		return obj.ToEntry()
	}
	prefixes := map[string]*ves_io_schema_aws_tgw_site.VPCIpPrefixesType{
		"vpc-1": {Prefixes: []string{"10.0.0.0/16"}},
	}
	req := &ves_io_schema_aws_tgw_site.SetVPCIpPrefixesRequest{Name: "tgw1", VpcIpPrefixes: prefixes}

	tests := []struct {
		name    string
		ctx     context.Context
		svc     *fakeTGWService
		wantErr string
	}{
		{name: "missing tenant", ctx: context.Background(), svc: &fakeTGWService{}, wantErr: "tenant missing in request"},
		{name: "find error", ctx: tenantCtx, svc: &fakeTGWService{findErr: errors.New("db down")}, wantErr: "cannot find aws_tgw_site entry tgw1"},
		{name: "not found", ctx: tenantCtx, svc: &fakeTGWService{}, wantErr: "aws_tgw_site tgw1 does not exist"},
		{name: "change error", ctx: tenantCtx, svc: &fakeTGWService{entry: newEntry(), found: true, chgErr: errors.New("conflict")}, wantErr: "could not update aws_tgw_site"},
		{name: "success", ctx: tenantCtx, svc: &fakeTGWService{entry: newEntry(), found: true}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rsp, err := (&tgwCustomAPIServer{sf: tc.svc}).SetVPCIpPrefixes(tc.ctx, req)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				assert.Nil(t, rsp)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, rsp)
			changed, ok := tc.svc.changed.(*ves_io_schema_aws_tgw_site.DBObject)
			require.True(t, ok)
			assert.Equal(t, prefixes, changed.Spec.GcSpec.VpcIpPrefixes)
		})
	}
}

// TestGetTestClientOpts verifies that the test client options use the test
// TLS bundle and set the server CN that the test fixtures expect.
func TestGetTestClientOpts(t *testing.T) {
	prevCN := vesapi.ServerCN
	t.Cleanup(func() { vesapi.ServerCN = prevCN })

	opts, err := getTestClientOpts("https://tenant.example.com/api", "acme", testProviderUserAgent())

	require.NoError(t, err)
	assert.Len(t, opts, 4)
	assert.Equal(t, "examplesvc.ves.io", vesapi.ServerCN)
}
