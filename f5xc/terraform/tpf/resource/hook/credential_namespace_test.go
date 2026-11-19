//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func apiCredentialSpecAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"password": types.StringType,
		"type": types.ListType{
			ElemType: types.StringType,
		},
		"virtual_k8s_name":      types.StringType,
		"virtual_k8s_namespace": types.StringType,
	}
}

func buildApiCredentialSpecObj(t *testing.T, credType string, vk8sNamespace types.String) types.Object {
	t.Helper()

	var typeList types.List
	if credType == "" {
		typeList = types.ListNull(types.StringType)
	} else {
		var diags interface{ HasError() bool }
		listVal, d := types.ListValue(types.StringType, []attr.Value{types.StringValue(credType)})
		diags = d
		if diags.HasError() {
			t.Fatalf("failed to create spec.type list: %v", d)
		}
		typeList = listVal
	}

	obj, diags := types.ObjectValue(
		apiCredentialSpecAttrTypes(),
		map[string]attr.Value{
			"password":              types.StringNull(),
			"type":                  typeList,
			"virtual_k8s_name":      types.StringValue("vk8s-cluster-1"),
			"virtual_k8s_namespace": vk8sNamespace,
		},
	)
	if diags.HasError() {
		t.Fatalf("failed to create api_credential spec object: %v", diags)
	}
	return obj
}

// buildApiCredentialSpecObjWithType builds an api_credential spec object with an
// explicit spec.type list value (for example, unknown or containing unknown elements).
func buildApiCredentialSpecObjWithType(t *testing.T, typeList types.List, vk8sNamespace types.String) types.Object {
	t.Helper()

	obj, diags := types.ObjectValue(
		apiCredentialSpecAttrTypes(),
		map[string]attr.Value{
			"password":              types.StringNull(),
			"type":                  typeList,
			"virtual_k8s_name":      types.StringValue("vk8s-cluster-1"),
			"virtual_k8s_namespace": vk8sNamespace,
		},
	)
	if diags.HasError() {
		t.Fatalf("failed to create api_credential spec object: %v", diags)
	}
	return obj
}

// mustStringList builds a known list of strings from the given element values.
func mustStringList(t *testing.T, elems ...attr.Value) types.List {
	t.Helper()

	list, diags := types.ListValue(types.StringType, elems)
	if diags.HasError() {
		t.Fatalf("failed to create string list: %v", diags)
	}
	return list
}

func serviceCredentialChoiceAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"api_certificate": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"password": types.StringType,
			},
		},
		"api_token": types.BoolType,
		"site_kubeconfig": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"site": types.StringType,
			},
		},
		"vk8s_kubeconfig": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"vk8s_cluster_name": types.StringType,
				"vk8s_namespace":    types.StringType,
			},
		},
	}
}

func buildServiceCredentialChoiceObj(t *testing.T, apiToken bool, vk8sObj types.Object) types.Object {
	t.Helper()

	apiCertAttrTypes := map[string]attr.Type{"password": types.StringType}
	siteKubeAttrTypes := map[string]attr.Type{"site": types.StringType}

	apiTokenVal := types.BoolNull()
	if apiToken {
		apiTokenVal = types.BoolValue(true)
	}

	obj, diags := types.ObjectValue(
		serviceCredentialChoiceAttrTypes(),
		map[string]attr.Value{
			"api_certificate": types.ObjectNull(apiCertAttrTypes),
			"api_token":       apiTokenVal,
			"site_kubeconfig": types.ObjectNull(siteKubeAttrTypes),
			"vk8s_kubeconfig": vk8sObj,
		},
	)
	if diags.HasError() {
		t.Fatalf("failed to create service_credential_choice object: %v", diags)
	}
	return obj
}

func TestResolveEffectiveNamespace(t *testing.T) {
	testCases := []struct {
		name                string
		configuredNamespace string
		vk8sNamespace       string
		want                string
	}{
		{
			name:                "prefers_vk8s_namespace_over_system",
			configuredNamespace: "system",
			vk8sNamespace:       "my-vk8s-ns",
			want:                "my-vk8s-ns",
		},
		{
			name:                "trims_vk8s_namespace",
			configuredNamespace: "system",
			vk8sNamespace:       "  app-ns  ",
			want:                "app-ns",
		},
		{
			name:                "falls_back_to_configured_namespace",
			configuredNamespace: "system",
			vk8sNamespace:       "",
			want:                "system",
		},
		{
			name:                "whitespace_vk8s_falls_back_to_configured",
			configuredNamespace: " custom-ns ",
			vk8sNamespace:       "   ",
			want:                "custom-ns",
		},
		{
			name:                "defaults_to_system_when_both_empty",
			configuredNamespace: "",
			vk8sNamespace:       "",
			want:                "system",
		},
		{
			name:                "whitespace_only_configured_defaults_to_system",
			configuredNamespace: "   ",
			vk8sNamespace:       "",
			want:                "system",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveEffectiveNamespace(tc.configuredNamespace, tc.vk8sNamespace); got != tc.want {
				t.Fatalf("ResolveEffectiveNamespace(%q, %q) = %q, want %q", tc.configuredNamespace, tc.vk8sNamespace, got, tc.want)
			}
		})
	}
}

func TestResolveApiCredentialNamespace(t *testing.T) {
	ctx := context.Background()
	specAttrTypes := apiCredentialSpecAttrTypes()

	testCases := []struct {
		name                string
		configuredNamespace types.String
		specObj             types.Object
		want                string
	}{
		{
			name:                "kube_config_uses_virtual_k8s_namespace",
			configuredNamespace: types.StringValue("system"),
			specObj:             buildApiCredentialSpecObj(t, "KUBE_CONFIG", types.StringValue("my-vk8s-ns")),
			want:                "my-vk8s-ns",
		},
		{
			name:                "kube_config_with_null_configured_namespace",
			configuredNamespace: types.StringNull(),
			specObj:             buildApiCredentialSpecObj(t, "KUBE_CONFIG", types.StringValue("my-vk8s-ns")),
			want:                "my-vk8s-ns",
		},
		{
			name:                "api_token_ignores_virtual_k8s_namespace",
			configuredNamespace: types.StringValue("system"),
			specObj:             buildApiCredentialSpecObj(t, "API_TOKEN", types.StringValue("ignored-ns")),
			want:                "system",
		},
		{
			name:                "api_certificate_uses_configured_namespace",
			configuredNamespace: types.StringValue("system"),
			specObj:             buildApiCredentialSpecObj(t, "API_CERTIFICATE", types.StringNull()),
			want:                "system",
		},
		{
			name:                "null_spec_uses_configured_namespace",
			configuredNamespace: types.StringValue("system"),
			specObj:             types.ObjectNull(specAttrTypes),
			want:                "system",
		},
		{
			name:                "unknown_spec_defaults_to_system_when_namespace_null",
			configuredNamespace: types.StringNull(),
			specObj:             types.ObjectUnknown(specAttrTypes),
			want:                "system",
		},
		{
			name:                "kube_config_with_unknown_vk8s_namespace_falls_back_to_system",
			configuredNamespace: types.StringValue("system"),
			specObj:             buildApiCredentialSpecObj(t, "KUBE_CONFIG", types.StringUnknown()),
			want:                "system",
		},
		{
			name:                "kube_config_type_with_surrounding_whitespace_is_trimmed",
			configuredNamespace: types.StringValue("system"),
			specObj:             buildApiCredentialSpecObj(t, " KUBE_CONFIG ", types.StringValue("my-vk8s-ns")),
			want:                "my-vk8s-ns",
		},
		{
			name:                "unknown_type_list_uses_virtual_k8s_namespace",
			configuredNamespace: types.StringValue("system"),
			specObj:             buildApiCredentialSpecObjWithType(t, types.ListUnknown(types.StringType), types.StringValue("my-vk8s-ns")),
			want:                "my-vk8s-ns",
		},
		{
			name:                "unknown_type_element_uses_virtual_k8s_namespace",
			configuredNamespace: types.StringValue("system"),
			specObj:             buildApiCredentialSpecObjWithType(t, mustStringList(t, types.StringUnknown()), types.StringValue("my-vk8s-ns")),
			want:                "my-vk8s-ns",
		},
		{
			name:                "whitespace_only_configured_namespace_defaults_to_system",
			configuredNamespace: types.StringValue("   "),
			specObj:             types.ObjectNull(specAttrTypes),
			want:                "system",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveApiCredentialNamespace(ctx, tc.configuredNamespace, tc.specObj); got != tc.want {
				t.Fatalf("ResolveApiCredentialNamespace() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveServiceCredentialNamespace(t *testing.T) {
	ctx := context.Background()
	vk8sAttrTypes := map[string]attr.Type{
		"vk8s_cluster_name": types.StringType,
		"vk8s_namespace":    types.StringType,
	}

	vk8sPopulated, diags := types.ObjectValue(vk8sAttrTypes, map[string]attr.Value{
		"vk8s_cluster_name": types.StringValue("vk8s-cluster-1"),
		"vk8s_namespace":    types.StringValue("svc-vk8s-ns"),
	})
	if diags.HasError() {
		t.Fatalf("failed to build vk8s_kubeconfig object: %v", diags)
	}

	vk8sEmptyNS, diags := types.ObjectValue(vk8sAttrTypes, map[string]attr.Value{
		"vk8s_cluster_name": types.StringValue("vk8s-cluster-1"),
		"vk8s_namespace":    types.StringNull(),
	})
	if diags.HasError() {
		t.Fatalf("failed to build vk8s_kubeconfig object with null ns: %v", diags)
	}

	vk8sUnknownNS, diags := types.ObjectValue(vk8sAttrTypes, map[string]attr.Value{
		"vk8s_cluster_name": types.StringValue("vk8s-cluster-1"),
		"vk8s_namespace":    types.StringUnknown(),
	})
	if diags.HasError() {
		t.Fatalf("failed to build vk8s_kubeconfig object with unknown ns: %v", diags)
	}

	vk8sPaddedNS, diags := types.ObjectValue(vk8sAttrTypes, map[string]attr.Value{
		"vk8s_cluster_name": types.StringValue("vk8s-cluster-1"),
		"vk8s_namespace":    types.StringValue("  svc-vk8s-ns  "),
	})
	if diags.HasError() {
		t.Fatalf("failed to build vk8s_kubeconfig object with padded ns: %v", diags)
	}

	testCases := []struct {
		name                string
		configuredNamespace types.String
		choiceObj           types.Object
		want                string
	}{
		{
			name:                "vk8s_kubeconfig_uses_vk8s_namespace",
			configuredNamespace: types.StringValue("system"),
			choiceObj:           buildServiceCredentialChoiceObj(t, false, vk8sPopulated),
			want:                "svc-vk8s-ns",
		},
		{
			name:                "vk8s_kubeconfig_with_null_configured_namespace",
			configuredNamespace: types.StringNull(),
			choiceObj:           buildServiceCredentialChoiceObj(t, false, vk8sPopulated),
			want:                "svc-vk8s-ns",
		},
		{
			name:                "api_token_choice_uses_system",
			configuredNamespace: types.StringValue("system"),
			choiceObj:           buildServiceCredentialChoiceObj(t, true, types.ObjectNull(vk8sAttrTypes)),
			want:                "system",
		},
		{
			name:                "vk8s_kubeconfig_with_null_vk8s_namespace_falls_back_to_system",
			configuredNamespace: types.StringValue("system"),
			choiceObj:           buildServiceCredentialChoiceObj(t, false, vk8sEmptyNS),
			want:                "system",
		},
		{
			name:                "null_choice_defaults_to_system",
			configuredNamespace: types.StringNull(),
			choiceObj:           types.ObjectNull(serviceCredentialChoiceAttrTypes()),
			want:                "system",
		},
		{
			name:                "unknown_choice_uses_configured_namespace",
			configuredNamespace: types.StringValue("system"),
			choiceObj:           types.ObjectUnknown(serviceCredentialChoiceAttrTypes()),
			want:                "system",
		},
		{
			name:                "unknown_vk8s_kubeconfig_uses_configured_namespace",
			configuredNamespace: types.StringValue("system"),
			choiceObj:           buildServiceCredentialChoiceObj(t, false, types.ObjectUnknown(vk8sAttrTypes)),
			want:                "system",
		},
		{
			name:                "vk8s_kubeconfig_with_unknown_vk8s_namespace_falls_back_to_system",
			configuredNamespace: types.StringValue("system"),
			choiceObj:           buildServiceCredentialChoiceObj(t, false, vk8sUnknownNS),
			want:                "system",
		},
		{
			name:                "vk8s_kubeconfig_namespace_is_trimmed",
			configuredNamespace: types.StringValue("system"),
			choiceObj:           buildServiceCredentialChoiceObj(t, false, vk8sPaddedNS),
			want:                "svc-vk8s-ns",
		},
		{
			name:                "whitespace_only_configured_namespace_defaults_to_system",
			configuredNamespace: types.StringValue("   "),
			choiceObj:           types.ObjectNull(serviceCredentialChoiceAttrTypes()),
			want:                "system",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveServiceCredentialNamespace(ctx, tc.configuredNamespace, tc.choiceObj); got != tc.want {
				t.Fatalf("ResolveServiceCredentialNamespace() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseCredentialImportID(t *testing.T) {
	testCases := []struct {
		id            string
		wantNamespace string
		wantVK8s      string
		wantName      string
		wantErr       bool
	}{
		{id: "system/cred-1", wantNamespace: "system", wantName: "cred-1"},
		{id: " system / cred-1 ", wantNamespace: "system", wantName: "cred-1"},
		{id: "system/ my-vk8s-ns /cred-1", wantNamespace: "system", wantVK8s: "my-vk8s-ns", wantName: "cred-1"},
		{id: "my-vk8s-ns/cred-1", wantNamespace: "system", wantVK8s: "my-vk8s-ns", wantName: "cred-1"},
		{id: "system/my-vk8s-ns/cred-1", wantNamespace: "system", wantVK8s: "my-vk8s-ns", wantName: "cred-1"},
		{id: "", wantErr: true},
		{id: "only-one", wantErr: true},
		{id: "/cred-1", wantErr: true},
		{id: "my-vk8s-ns/", wantErr: true},
		{id: "system//cred-1", wantErr: true},
		{id: "/my-vk8s-ns/cred-1", wantErr: true},
		{id: "system/my-vk8s-ns/", wantErr: true},
		{id: "other/my-vk8s-ns/cred-1", wantErr: true},
		{id: "a/b/c/d", wantErr: true},
		{id: " /cred-1", wantErr: true},
		{id: "my-vk8s-ns/  ", wantErr: true},
		{id: "system/ /cred-1", wantErr: true},
		{id: "system/my-vk8s-ns/ ", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.id, func(t *testing.T) {
			ns, vk8s, name, err := parseCredentialImportID(tc.id)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseCredentialImportID(%q) expected error, got (%q, %q, %q)", tc.id, ns, vk8s, name)
				}
				if !strings.HasPrefix(err.Error(), "expected import ID format") {
					t.Fatalf("error %q must start with a lowercase \"expected import ID format\"", err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCredentialImportID(%q) unexpected error: %v", tc.id, err)
			}
			if ns != tc.wantNamespace || vk8s != tc.wantVK8s || name != tc.wantName {
				t.Fatalf("parseCredentialImportID(%q) = (%q, %q, %q), want (%q, %q, %q)", tc.id, ns, vk8s, name, tc.wantNamespace, tc.wantVK8s, tc.wantName)
			}
		})
	}
}

func TestBuildNestedObject(t *testing.T) {
	leafObjType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"vk8s_namespace": types.StringType,
		"count":          types.Int64Type,
	}}
	rootType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"vk8s_kubeconfig": leafObjType,
		"api_token":       types.BoolType,
		"labels":          types.MapType{ElemType: types.StringType},
		"names":           types.ListType{ElemType: types.StringType},
		"tags":            types.SetType{ElemType: types.StringType},
		"ratio":           types.Float64Type,
		"big":             types.NumberType,
	}}

	t.Run("sets_leaf_and_nulls_every_other_attribute", func(t *testing.T) {
		obj, diags := buildNestedObject(rootType, "choice", []string{"vk8s_kubeconfig", "vk8s_namespace"}, "my-ns")
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		attrs := obj.Attributes()
		for name, v := range attrs {
			if name == "vk8s_kubeconfig" {
				continue
			}
			if !v.IsNull() {
				t.Fatalf("attribute %q = %v, want null", name, v)
			}
		}
		child := attrs["vk8s_kubeconfig"].(types.Object)
		if got := child.Attributes()["vk8s_namespace"].(types.String).ValueString(); got != "my-ns" {
			t.Fatalf("vk8s_namespace = %q, want my-ns", got)
		}
		if !child.Attributes()["count"].IsNull() {
			t.Fatalf("count = %v, want null", child.Attributes()["count"])
		}
	})

	errorCases := []struct {
		name     string
		leafPath []string
	}{
		{name: "empty_leaf_path", leafPath: nil},
		{name: "missing_attribute", leafPath: []string{"does_not_exist"}},
		{name: "leaf_is_not_string", leafPath: []string{"api_token"}},
		{name: "intermediate_is_not_object", leafPath: []string{"api_token", "x"}},
		{name: "missing_nested_attribute", leafPath: []string{"vk8s_kubeconfig", "does_not_exist"}},
		{name: "nested_leaf_is_not_string", leafPath: []string{"vk8s_kubeconfig", "count"}},
	}
	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			obj, diags := buildNestedObject(rootType, "choice", tc.leafPath, "my-ns")
			if !diags.HasError() {
				t.Fatalf("expected diagnostics for leafPath %v", tc.leafPath)
			}
			if !obj.IsNull() {
				t.Fatalf("expected a null object on error, got %v", obj)
			}
		})
	}
}

func TestNestedObjectWithString_Errors(t *testing.T) {
	ctx := context.Background()

	t.Run("nil_schema", func(t *testing.T) {
		_, diags := nestedObjectWithString(ctx, nil, "spec", []string{"virtual_k8s_namespace"}, "ns")
		if !diags.HasError() {
			t.Fatal("expected an error for a nil schema")
		}
	})

	s := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"namespace": schema.StringAttribute{Required: true},
		},
	}

	t.Run("missing_root_attribute", func(t *testing.T) {
		_, diags := nestedObjectWithString(ctx, s, "spec", []string{"virtual_k8s_namespace"}, "ns")
		if !diags.HasError() {
			t.Fatal("expected an error when the root attribute does not exist")
		}
	})

	t.Run("root_attribute_is_not_object", func(t *testing.T) {
		_, diags := nestedObjectWithString(ctx, s, "namespace", []string{"x"}, "ns")
		if !diags.HasError() {
			t.Fatal("expected an error when the root attribute is not an object")
		}
	})
}

func TestNullValueOf_NilType(t *testing.T) {
	v, err := nullValueOf(nil)
	if err == nil {
		t.Fatalf("nullValueOf(nil) = %v, want an error", v)
	}
}

func TestBuildNestedObject_NilAttributeType(t *testing.T) {
	objType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"vk8s_namespace": types.StringType,
		"broken":         nil,
	}}
	obj, diags := buildNestedObject(objType, "spec", []string{"vk8s_namespace"}, "my-ns")
	if !diags.HasError() {
		t.Fatal("expected diagnostics when a sibling attribute type is nil")
	}
	if !obj.IsNull() {
		t.Fatalf("expected a null object on error, got %v", obj)
	}
}

func TestNullValueOf(t *testing.T) {
	testCases := []attr.Type{
		types.StringType,
		types.BoolType,
		types.Int64Type,
		types.Float64Type,
		types.NumberType,
		types.Int32Type,
		types.ObjectType{AttrTypes: map[string]attr.Type{"a": types.StringType}},
		types.ListType{ElemType: types.StringType},
		types.SetType{ElemType: types.StringType},
		types.MapType{ElemType: types.StringType},
	}
	for _, typ := range testCases {
		t.Run(typ.String(), func(t *testing.T) {
			v, err := nullValueOf(typ)
			if err != nil {
				t.Fatalf("nullValueOf(%s) returned error: %v", typ, err)
			}
			if v == nil || !v.IsNull() {
				t.Fatalf("nullValueOf(%s) = %v, want a null value", typ, v)
			}
			if !v.Type(context.Background()).Equal(typ) {
				t.Fatalf("nullValueOf(%s) has type %s", typ, v.Type(context.Background()))
			}
		})
	}
}
