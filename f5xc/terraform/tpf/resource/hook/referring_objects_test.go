//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseReferringObjects_EdgeCases verifies marker handling, malformed paths,
// and the backward-compatible alias. TestParseReferringObjects covers the common
// multi-referrer and tenant-prefix cases.
func TestParseReferringObjects_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		errText string
		want    []ReferringResource
	}{
		{
			name:    "empty text",
			errText: "",
			want:    nil,
		},
		{
			name:    "no schema object in text",
			errText: "rpc error: code = Internal desc = boom",
			want:    nil,
		},
		{
			name: "object before marker is ignored",
			errText: "DeleteResource: ves.io.schema.forward_proxy_policy.Object tenant/default/fpp1 " +
				"is still being referred by 1 objects - [ves.io.schema.views.origin_pool.Object default/op1]",
			want: []ReferringResource{{Kind: "origin_pools", Namespace: "default", Name: "op1"}},
		},
		{
			name: "whole text is parsed without marker",
			errText: "ves.io.schema.virtual_network.Object system/vn1 " +
				"ves.io.schema.views.aws_vpc_site.Object system/site1",
			want: []ReferringResource{
				{Kind: "virtual_networks", Namespace: "system", Name: "vn1"},
				{Kind: "aws_vpc_sites", Namespace: "system", Name: "site1"},
			},
		},
		{
			name:    "trailing bracket is not part of the name",
			errText: "is still being referred by 1 objects - [ves.io.schema.views.tcp_loadbalancer.Object ns2/lb2]",
			want:    []ReferringResource{{Kind: "tcp_loadbalancers", Namespace: "ns2", Name: "lb2"}},
		},
		{
			name:    "path without namespace is skipped",
			errText: "is still being referred by 1 objects - [ves.io.schema.views.origin_pool.Object op1]",
			want:    nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseReferringObjects(tc.errText)

			assert.Equal(t, tc.want, got)
			assert.Equal(t, got, ParseReferringHTTPLoadBalancers(tc.errText), "alias must match ParseReferringObjects")
		})
	}
}

// TestReferrerKindsContains verifies exact, case-sensitive membership.
func TestReferrerKindsContains(t *testing.T) {
	kinds := referrerKinds{"origin_pools", "http_loadbalancers"}

	tests := []struct {
		kind string
		want bool
	}{
		{kind: "origin_pools", want: true},
		{kind: "http_loadbalancers", want: true},
		{kind: "origin_pool", want: false},
		{kind: "Origin_pools", want: false},
		{kind: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.kind, func(t *testing.T) {
			assert.Equal(t, tc.want, kinds.contains(tc.kind))
		})
	}

	assert.False(t, referrerKinds(nil).contains("origin_pools"), "nil set contains nothing")
}

// TestPolicyReferrerKindFilters verifies the parent kinds each policy can be
// unbound from, and that each policy's own plural kind is rejected.
func TestPolicyReferrerKindFilters(t *testing.T) {
	tests := []struct {
		name   string
		filter func(string) bool
		accept []string
		reject []string
	}{
		{
			name:   "enhanced_firewall_policy",
			filter: isEnhancedFirewallPolicyReferrerKind,
			accept: []string{
				"appstack_sites", "aws_vpc_sites", "azure_vnet_sites", "gcp_vpc_sites",
				"aws_tgw_sites", "securemesh_sites", "virtual_networks",
			},
			reject: []string{"enhanced_firewall_policys", "network_policy_sets", "http_loadbalancers", ""},
		},
		{
			name:   "forward_proxy_policy",
			filter: isForwardProxyPolicyReferrerKind,
			accept: []string{"origin_pools", "http_loadbalancers", "tcp_loadbalancers", "proxys"},
			reject: []string{"forward_proxy_policys", "virtual_networks", "aws_vpc_sites", ""},
		},
		{
			name:   "network_policy",
			filter: isNetworkPolicyReferrerKind,
			accept: []string{
				"virtual_networks", "network_policy_sets", "appstack_sites", "aws_vpc_sites",
				"azure_vnet_sites", "gcp_vpc_sites", "aws_tgw_sites", "securemesh_sites",
			},
			reject: []string{"network_policys", "http_loadbalancers", "origin_pools", ""},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, kind := range tc.accept {
				assert.True(t, tc.filter(kind), "expected %q to be accepted", kind)
			}
			for _, kind := range tc.reject {
				assert.False(t, tc.filter(kind), "expected %q to be rejected", kind)
			}
		})
	}
}

// A referrer path without a namespace segment is ignored.
func TestParseReferringObjects_SkipsPathWithoutNamespace(t *testing.T) {
	refs := ParseReferringObjects("is still being referred by 1 objects - [ves.io.schema.views.http_loadbalancer.Object lb1]")
	assert.Empty(t, refs)
}
