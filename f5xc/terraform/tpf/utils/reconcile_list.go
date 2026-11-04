//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package utils

import (
	"context"
	"net/netip"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ReconcileListForApply reconciles a list returned by the backend with the planned list.
//
// Why: The F5 XC backend frequently normalizes or sorts list elements (such as IP prefixes,
// port numbers, HTTP status codes, or BGP ASNs) upon persistence. If the provider writes
// the server's reordered list directly into state, Terraform Core fails during apply with
// "Provider produced inconsistent result after apply".
//
// How: If the planned and API lists are identical as multisets (Set(planned) == Set(api) via
// attr.Value.Equal) and the planned list contains no unknown values, the planned list is
// preserved directly in state. If drift has occurred (elements added or deleted out-of-band),
// existing planned elements retain their planned order while newly added elements are appended
// at the end.
//
// Parameters:
//   - ctx: context.Context - context for element type inspection and diagnostics
//   - planned: types.List - the planned list from request.Plan
//   - api: types.List - the list constructed from the backend API response
//
// Returns:
//   - types.List: the reconciled list safe to store in Terraform state
//   - diag.Diagnostics: diagnostics collected during list construction
//
// Example:
//
//	reconciled, diags := utils.ReconcileListForApply(ctx, plannedList, apiList)
//	response.Diagnostics.Append(diags...)
func ReconcileListForApply(ctx context.Context, planned, api types.List) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if planned.IsNull() || planned.IsUnknown() {
		return api, diags
	}
	if api.IsNull() || api.IsUnknown() {
		return planned, diags
	}
	if !ValueContainsUnknown(planned) {
		if listsEqualAsSets(planned.Elements(), api.Elements()) {
			return planned, diags
		}
	}

	return reconcileOrderGeneric(ctx, planned, api)
}

// ReconcileListForRead reconciles a list returned by the backend with the prior state list.
//
// Why: When reading a resource from the backend, a pure permutation caused by server-side sorting
// must not trigger spurious plan drift. However, legitimate out-of-band additions or deletions
// must still be surfaced as Terraform drift.
//
// How: If the prior state and API lists contain the exact same elements as a set, the prior state
// order is retained. If elements were added or removed in the backend, prior elements retain their
// order while new elements are appended at the end.
//
// Parameters:
//   - ctx: context.Context - context for element type inspection and diagnostics
//   - prior: types.List - the list from request.State (prior state)
//   - api: types.List - the list constructed from the backend API response
//
// Returns:
//   - types.List: the reconciled list safe to store in Terraform state
//   - diag.Diagnostics: diagnostics collected during list construction
//
// Example:
//
//	reconciled, diags := utils.ReconcileListForRead(ctx, priorList, apiList)
//	response.Diagnostics.Append(diags...)
func ReconcileListForRead(ctx context.Context, prior, api types.List) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if prior.IsNull() || prior.IsUnknown() {
		return api, diags
	}
	if api.IsNull() || api.IsUnknown() {
		return prior, diags
	}

	if listsEqualAsSets(prior.Elements(), api.Elements()) {
		return prior, diags
	}

	return reconcileOrderGeneric(ctx, prior, api)
}

// listsEqualAsSets determines whether two slices of attr.Value contain identical elements as a multiset.
//
// Why: Enables order-agnostic equality checking across any scalar type (strings, integers, floats, enums)
// using the framework's native attr.Value.Equal method.
//
// How: Checks equal slice lengths first. Iterates through slice a, finding and claiming a matching,
// unconsumed element in slice b. Returns false if lengths differ or any element in a lacks a match in b.
//
// Parameters:
//   - a: []attr.Value - first element slice
//   - b: []attr.Value - second element slice
//
// Returns:
//   - bool: true if both slices contain the same elements with equal multiplicities, false otherwise
//
// Example:
//
//	equal := listsEqualAsSets([]attr.Value{types.StringValue("b"), types.StringValue("a")}, []attr.Value{types.StringValue("a"), types.StringValue("b")})
//	// equal == true
func listsEqualAsSets(a, b []attr.Value) bool {
	if len(a) != len(b) {
		return false
	}
	matched := make([]bool, len(b))
	for _, elemA := range a {
		found := false
		for j, elemB := range b {
			if !matched[j] && elementsSemanticallyEqual(elemA, elemB) {
				matched[j] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// elementsSemanticallyEqual compares two attr.Value elements, treating canonicalized IP
// addresses and CIDR prefixes (such as "2001:db8:1:0:0:0:0:0/64" and "2001:db8:1::/64")
// as equal.
//
// Why: The F5 XC backend canonicalizes IPv6 addresses and CIDR prefixes according to RFC 5952
// (e.g. compressing zero runs and lowercasing hex digits). Literal string comparison treats
// the compressed form as a changed value and causes "Provider produced inconsistent result after apply".
//
// How: First checks exact equality via elemA.Equal(elemB). If both values are known non-null
// types.String values, delegates to ipStringsOrPrefixesEqual to compare them as netip.Prefix
// or netip.Addr.
//
// Parameters:
//   - elemA: attr.Value - first element (typically from planned or prior state)
//   - elemB: attr.Value - second element (typically from API response)
//
// Returns:
//   - bool: true if the elements are equal literally or represent the same IP/CIDR
//
// Example:
//
//	eq := elementsSemanticallyEqual(types.StringValue("2001:db8:1:0:0:0:0:0/64"), types.StringValue("2001:db8:1::/64"))
//	// eq == true
func elementsSemanticallyEqual(elemA, elemB attr.Value) bool {
	if elemA.Equal(elemB) {
		return true
	}
	strA, okA := elemA.(types.String)
	strB, okB := elemB.(types.String)
	if !okA || !okB || strA.IsNull() || strA.IsUnknown() || strB.IsNull() || strB.IsUnknown() {
		return false
	}
	return ipStringsOrPrefixesEqual(strA.ValueString(), strB.ValueString())
}

// ipStringsOrPrefixesEqual checks whether two strings parse to the same canonical CIDR prefix
// or IP address.
//
// Why: Isolates netip parsing logic for CIDR prefixes and standalone IP addresses so non-IP
// strings quickly return false while equivalent IPv4/IPv6 representations match.
//
// How: Attempts netip.ParsePrefix on both strings and compares Masked() prefixes (ensuring both
// address bits and prefix length match). If prefix parsing fails, attempts netip.ParseAddr on
// both strings and compares canonical addresses.
//
// Parameters:
//   - a: string - first string candidate
//   - b: string - second string candidate
//
// Returns:
//   - bool: true if both strings are valid CIDR prefixes or IP addresses representing the same network/host
//
// Example:
//
//	eq := ipStringsOrPrefixesEqual("2001:db8:1:0:0:0:0:0/64", "2001:db8:1::/64")
//	// eq == true
func ipStringsOrPrefixesEqual(a, b string) bool {
	if pA, errA := netip.ParsePrefix(a); errA == nil {
		if pB, errB := netip.ParsePrefix(b); errB == nil {
			return pA.Masked() == pB.Masked() && pA.Addr().Compare(pB.Addr()) == 0
		}
		return false
	}
	if addrA, errA := netip.ParseAddr(a); errA == nil {
		if addrB, errB := netip.ParseAddr(b); errB == nil {
			return addrA.Compare(addrB) == 0
		}
	}
	return false
}

// reconcileOrderGeneric merges base and api lists, preserving base ordering for shared elements.
//
// Why: When the API returns a modified list (drift), elements configured by the user should keep
// their expected relative position, while newly discovered elements from the backend appear at the end.
//
// How: Iterates through base elements; for each element still present in the API, appends it to the result
// and marks it as matched. Then appends any remaining unmatched API elements. Constructs a new types.List
// with the API's element type.
//
// Parameters:
//   - ctx: context.Context - context for ElementType determination
//   - base: types.List - baseline list (planned or prior state)
//   - api: types.List - live API list from backend
//
// Returns:
//   - types.List: reconstructed list combining base ordering and live values
//   - diag.Diagnostics: diagnostics from types.ListValue creation
//
// Example:
//
//	merged, diags := reconcileOrderGeneric(ctx, plannedList, apiList)
func reconcileOrderGeneric(ctx context.Context, base, api types.List) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	apiElements := api.Elements()
	baseElements := base.Elements()

	mergedElements := make([]attr.Value, 0, len(apiElements))
	matchedAPI := make([]bool, len(apiElements))

	// 1. Elements from base that still exist in the API retain their base ordering and spelling.
	for _, baseElem := range baseElements {
		if baseElem.IsNull() || baseElem.IsUnknown() {
			continue
		}
		for j, apiElem := range apiElements {
			if !matchedAPI[j] && elementsSemanticallyEqual(baseElem, apiElem) {
				mergedElements = append(mergedElements, baseElem)
				matchedAPI[j] = true
				break
			}
		}
	}

	// 2. Newly added elements from the backend are appended at the end.
	for j, apiElem := range apiElements {
		if !matchedAPI[j] {
			mergedElements = append(mergedElements, apiElem)
		}
	}

	reconciledList, listDiags := types.ListValue(api.ElementType(ctx), mergedElements)
	diags.Append(listDiags...)
	return reconciledList, diags
}
