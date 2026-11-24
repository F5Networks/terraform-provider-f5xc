//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test replaces per-resource schema tests for generated code. It checks
// every registered resource and data source for invariants that
// ValidateImplementation does not cover. Terraform checks the
// Required/Optional/Computed rules only when it validates a user config, so a
// broken flag combination is not caught without this test.

// schemaAttribute is the subset of the framework attribute interface that the
// invariant checks use. Resource and data source attributes both satisfy it.
type schemaAttribute interface {
	IsRequired() bool
	IsOptional() bool
	IsComputed() bool
	IsWriteOnly() bool
	GetDescription() string
	GetMarkdownDescription() string
}

// dataSourceLookupKeys are the only top-level attributes a data source may
// require. Every other input must be optional so a lookup by name works.
var dataSourceLookupKeys = map[string]bool{"name": true, "namespace": true}

// schemaWalkResult collects the findings of one schema walk.
type schemaWalkResult struct {
	violations    []string // hard invariant failures, as "path: reason"
	noDescription []string // attribute paths without a description
}

// toAnyMap converts a schema attribute or block map to map[string]any. The ~
// constraint also accepts the framework's internal named map types.
func toAnyMap[M ~map[string]V, V any](m M) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// nestedAttribute matches resource and data source nested attributes, which
// share the same framework interface.
type nestedAttribute = rschema.NestedAttribute

// nestedBlock matches resource and data source blocks, which share the same
// framework interface.
type nestedBlock = rschema.Block

// walkSchema checks every attribute under attrs and blocks, recursing into
// nested attributes and nested blocks. prefix is the dotted path of the parent.
func walkSchema(prefix string, attrs, blocks map[string]any, res *schemaWalkResult) {
	for name, raw := range attrs {
		path := prefix + name
		a, ok := raw.(schemaAttribute)
		if !ok {
			res.violations = append(res.violations, fmt.Sprintf("%s: unsupported attribute type %T", path, raw))
			continue
		}
		checkAttributeFlags(path, a, res)

		if nested, ok := raw.(nestedAttribute); ok {
			walkSchema(path+".", toAnyMap(nested.GetNestedObject().GetAttributes()), nil, res)
		}
	}

	for name, raw := range blocks {
		path := prefix + name
		b, ok := raw.(nestedBlock)
		if !ok {
			res.violations = append(res.violations, fmt.Sprintf("%s: unsupported block type %T", path, raw))
			continue
		}
		obj := b.GetNestedObject()
		walkSchema(path+".", toAnyMap(obj.GetAttributes()), toAnyMap(obj.GetBlocks()), res)
	}
}

// checkAttributeFlags records violations of the Required/Optional/Computed and
// write-only rules that Terraform enforces at config validation time.
func checkAttributeFlags(path string, a schemaAttribute, res *schemaWalkResult) {
	req, opt, comp, wo := a.IsRequired(), a.IsOptional(), a.IsComputed(), a.IsWriteOnly()

	switch {
	case !req && !opt && !comp:
		res.violations = append(res.violations, path+": none of Required, Optional, or Computed is set")
	case req && opt:
		res.violations = append(res.violations, path+": Required and Optional are both set")
	case req && comp:
		res.violations = append(res.violations, path+": Required and Computed are both set")
	}
	if wo && comp {
		res.violations = append(res.violations, path+": WriteOnly attribute is Computed")
	}

	if a.GetDescription() == "" && a.GetMarkdownDescription() == "" {
		res.noDescription = append(res.noDescription, path)
	}
}

func providerTypeName(t *testing.T, p *F5XCProvider) string {
	t.Helper()
	var resp provider.MetadataResponse
	p.Metadata(context.Background(), provider.MetadataRequest{}, &resp)
	require.NotEmpty(t, resp.TypeName)
	return resp.TypeName
}

// descriptionReport counts attributes without a description. A missing
// description gives an empty entry in the generated docs, but it does not
// break the provider, so it is logged and not a failure.
type descriptionReport struct {
	attrs   int
	schemas map[string]int
}

func (r *descriptionReport) add(typeName string, paths []string) {
	if len(paths) == 0 {
		return
	}
	r.attrs += len(paths)
	r.schemas[typeName] = len(paths)
}

func (r *descriptionReport) log(t *testing.T, kind string) {
	t.Helper()
	if r.attrs == 0 {
		return
	}
	names := make([]string, 0, len(r.schemas))
	for name := range r.schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) > 10 {
		names = append(names[:10], "...")
	}
	t.Logf("%d %s attributes in %d schemas have no description (for example: %s)",
		r.attrs, kind, len(r.schemas), strings.Join(names, ", "))
}

// TestProviderResources_SchemaSmoke verifies that every registered resource has
// a unique, provider-prefixed type name and a schema that builds without
// panics, passes framework implementation validation, and sets a valid
// Required/Optional/Computed combination on every attribute.
func TestProviderResources_SchemaSmoke(t *testing.T) {
	ctx := context.Background()
	p := &F5XCProvider{}
	providerName := providerTypeName(t, p)

	factories := p.Resources(ctx)
	require.NotEmpty(t, factories)

	seen := make(map[string]bool, len(factories))
	report := &descriptionReport{schemas: map[string]int{}}
	for _, newResource := range factories {
		r := newResource()

		var meta resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: providerName}, &meta)
		typeName := meta.TypeName

		t.Run(typeName, func(t *testing.T) {
			require.True(t, strings.HasPrefix(typeName, providerName+"_"), "type name %q must start with %q", typeName, providerName+"_")
			require.False(t, seen[typeName], "duplicate resource type name %q", typeName)
			seen[typeName] = true

			var resp resource.SchemaResponse
			require.NotPanics(t, func() { r.Schema(ctx, resource.SchemaRequest{}, &resp) })
			require.False(t, resp.Diagnostics.HasError(), "Schema() diagnostics: %v", resp.Diagnostics)
			assert.NotEmpty(t, resp.Schema.Attributes, "schema must define attributes")

			diags := resp.Schema.ValidateImplementation(ctx)
			assert.False(t, diags.HasError(), "ValidateImplementation diagnostics: %v", diags)

			var res schemaWalkResult
			walkSchema("", toAnyMap(resp.Schema.Attributes), toAnyMap(resp.Schema.Blocks), &res)
			sort.Strings(res.violations)
			assert.Empty(t, res.violations, "schema invariant violations")
			report.add(typeName, res.noDescription)
		})
	}
	report.log(t, "resource")
}

// TestProviderDataSources_SchemaSmoke verifies the same invariants for every
// registered data source, and that only lookup keys are required at the top level.
func TestProviderDataSources_SchemaSmoke(t *testing.T) {
	ctx := context.Background()
	p := &F5XCProvider{}
	providerName := providerTypeName(t, p)

	factories := p.DataSources(ctx)
	require.NotEmpty(t, factories)

	seen := make(map[string]bool, len(factories))
	report := &descriptionReport{schemas: map[string]int{}}
	for _, newDataSource := range factories {
		d := newDataSource()

		var meta datasource.MetadataResponse
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: providerName}, &meta)
		typeName := meta.TypeName

		t.Run(typeName, func(t *testing.T) {
			require.True(t, strings.HasPrefix(typeName, providerName+"_"), "type name %q must start with %q", typeName, providerName+"_")
			require.False(t, seen[typeName], "duplicate data source type name %q", typeName)
			seen[typeName] = true

			var resp datasource.SchemaResponse
			require.NotPanics(t, func() { d.Schema(ctx, datasource.SchemaRequest{}, &resp) })
			require.False(t, resp.Diagnostics.HasError(), "Schema() diagnostics: %v", resp.Diagnostics)
			assert.NotEmpty(t, resp.Schema.Attributes, "schema must define attributes")

			diags := resp.Schema.ValidateImplementation(ctx)
			assert.False(t, diags.HasError(), "ValidateImplementation diagnostics: %v", diags)

			var res schemaWalkResult
			walkSchema("", toAnyMap(resp.Schema.Attributes), toAnyMap(resp.Schema.Blocks), &res)
			sort.Strings(res.violations)
			assert.Empty(t, res.violations, "schema invariant violations")
			report.add(typeName, res.noDescription)

			for name, a := range resp.Schema.Attributes {
				if a.IsRequired() {
					assert.True(t, dataSourceLookupKeys[name], "top-level attribute %q is required but is not a lookup key", name)
				}
			}
		})
	}
	report.log(t, "data source")
}

// TestWalkSchema_DetectsViolations verifies that the walker reports each broken
// flag combination with its full path, including nested attributes and blocks.
func TestWalkSchema_DetectsViolations(t *testing.T) {
	attrs := map[string]rschema.Attribute{
		"ok":            rschema.StringAttribute{Optional: true, Description: "d"},
		"no_flags":      rschema.StringAttribute{Description: "d"},
		"req_opt":       rschema.StringAttribute{Required: true, Optional: true, Description: "d"},
		"req_comp":      rschema.StringAttribute{Required: true, Computed: true, Description: "d"},
		"wo_comp":       rschema.StringAttribute{WriteOnly: true, Optional: true, Computed: true, Description: "d"},
		"no_desc":       rschema.StringAttribute{Computed: true},
		"nested_holder": rschema.SingleNestedAttribute{Optional: true, Description: "d", Attributes: map[string]rschema.Attribute{"inner": rschema.BoolAttribute{Description: "d"}}},
	}
	blocks := map[string]rschema.Block{
		"blk": rschema.SingleNestedBlock{Attributes: map[string]rschema.Attribute{"field": rschema.StringAttribute{Required: true, Computed: true, Description: "d"}}},
	}

	var res schemaWalkResult
	walkSchema("", toAnyMap(attrs), toAnyMap(blocks), &res)
	sort.Strings(res.violations)

	assert.Equal(t, []string{
		"blk.field: Required and Computed are both set",
		"nested_holder.inner: none of Required, Optional, or Computed is set",
		"no_flags: none of Required, Optional, or Computed is set",
		"req_comp: Required and Computed are both set",
		"req_opt: Required and Optional are both set",
		"wo_comp: WriteOnly attribute is Computed",
	}, res.violations)
	assert.Equal(t, []string{"no_desc"}, res.noDescription)
}

// TestWalkSchema_DataSourceNested verifies that the walker also recurses into
// data source nested attributes and blocks.
func TestWalkSchema_DataSourceNested(t *testing.T) {
	attrs := map[string]dsschema.Attribute{
		"list": dsschema.ListNestedAttribute{
			Computed:    true,
			Description: "d",
			NestedObject: dsschema.NestedAttributeObject{Attributes: map[string]dsschema.Attribute{
				"bad": dsschema.StringAttribute{Description: "d"},
			}},
		},
	}
	blocks := map[string]dsschema.Block{
		"blk": dsschema.ListNestedBlock{NestedObject: dsschema.NestedBlockObject{Attributes: map[string]dsschema.Attribute{
			"bad": dsschema.StringAttribute{Required: true, Optional: true, Description: "d"},
		}}},
	}

	var res schemaWalkResult
	walkSchema("", toAnyMap(attrs), toAnyMap(blocks), &res)
	sort.Strings(res.violations)

	assert.Equal(t, []string{
		"blk.bad: Required and Optional are both set",
		"list.bad: none of Required, Optional, or Computed is set",
	}, res.violations)
}
