//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

package requiredifvalidator_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/validators/requiredifvalidator"
)

// buildConfig creates a tfsdk.Config with two string attributes: "type" and "site_name".
// Pass "" to set a field to null.
func buildConfig(t *testing.T, siblingVal string, currentVal string) tfsdk.Config {
	t.Helper()

	s := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"type":      schema.StringAttribute{Optional: true},
			"site_name": schema.StringAttribute{Optional: true},
		},
	}

	rawType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"type":      tftypes.String,
			"site_name": tftypes.String,
		},
	}

	var typeVal, siteVal interface{}
	if siblingVal != "" {
		typeVal = siblingVal
	}
	if currentVal != "" {
		siteVal = currentVal
	}

	raw := tftypes.NewValue(rawType, map[string]tftypes.Value{
		"type":      tftypes.NewValue(tftypes.String, typeVal),
		"site_name": tftypes.NewValue(tftypes.String, siteVal),
	})

	return tfsdk.Config{
		Raw:    raw,
		Schema: s,
	}
}

func TestRequiredIf_SiblingMatches_CurrentSet(t *testing.T) {
	t.Parallel()

	v := requiredifvalidator.RequiredIf("type", "JWT")
	cfg := buildConfig(t, "JWT", "my-site")

	req := validator.StringRequest{
		Path:        path.Root("site_name"),
		Config:      cfg,
		ConfigValue: types.StringValue("my-site"),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error when sibling matches and current is set, got: %s", resp.Diagnostics)
	}
}

func TestRequiredIf_SiblingMatches_CurrentNull(t *testing.T) {
	t.Parallel()

	v := requiredifvalidator.RequiredIf("type", "JWT")
	cfg := buildConfig(t, "JWT", "")

	req := validator.StringRequest{
		Path:        path.Root("site_name"),
		Config:      cfg,
		ConfigValue: types.StringNull(),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error when sibling matches and current is null, got none")
	}
}

func TestRequiredIf_SiblingDoesNotMatch_CurrentNull(t *testing.T) {
	t.Parallel()

	v := requiredifvalidator.RequiredIf("type", "JWT")
	cfg := buildConfig(t, "BASIC", "")

	req := validator.StringRequest{
		Path:        path.Root("site_name"),
		Config:      cfg,
		ConfigValue: types.StringNull(),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error when sibling does not match, got: %s", resp.Diagnostics)
	}
}

func TestRequiredIf_SiblingNull_CurrentNull(t *testing.T) {
	t.Parallel()

	v := requiredifvalidator.RequiredIf("type", "JWT")
	cfg := buildConfig(t, "", "")

	req := validator.StringRequest{
		Path:        path.Root("site_name"),
		Config:      cfg,
		ConfigValue: types.StringNull(),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error when sibling is null, got: %s", resp.Diagnostics)
	}
}

func TestRequiredIf_CurrentUnknown_Skipped(t *testing.T) {
	t.Parallel()

	v := requiredifvalidator.RequiredIf("type", "JWT")
	cfg := buildConfig(t, "JWT", "")

	req := validator.StringRequest{
		Path:        path.Root("site_name"),
		Config:      cfg,
		ConfigValue: types.StringUnknown(),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	// Unknown current value — validation is skipped
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for unknown current value, got: %s", resp.Diagnostics)
	}
}

func TestRequiredIf_MultipleSiblingValues_Matches(t *testing.T) {
	t.Parallel()

	v := requiredifvalidator.RequiredIf("type", "JWT", "OAUTH2")
	cfg := buildConfig(t, "OAUTH2", "")

	req := validator.StringRequest{
		Path:        path.Root("site_name"),
		Config:      cfg,
		ConfigValue: types.StringNull(),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error when sibling matches one of multiple values and current is null, got none")
	}
}

func TestRequiredIf_Description(t *testing.T) {
	t.Parallel()

	v := requiredifvalidator.RequiredIf("type", "JWT")
	desc := v.Description(context.Background())
	if desc == "" {
		t.Error("expected non-empty description")
	}
}

func TestRequiredIf_MarkdownDescription(t *testing.T) {
	t.Parallel()

	v := requiredifvalidator.RequiredIf("type", "JWT")
	desc := v.MarkdownDescription(context.Background())
	if desc == "" {
		t.Error("expected non-empty markdown description")
	}
}

// buildSiblingConfig returns a config with a "site_name" string attribute and
// a "type" sibling of the given schema attribute and raw value.
func buildSiblingConfig(t *testing.T, sibling schema.Attribute, siblingType tftypes.Type, siblingVal tftypes.Value) tfsdk.Config {
	t.Helper()

	rawType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"type":      siblingType,
		"site_name": tftypes.String,
	}}
	return tfsdk.Config{
		Schema: schema.Schema{Attributes: map[string]schema.Attribute{
			"type":      sibling,
			"site_name": schema.StringAttribute{Optional: true},
		}},
		Raw: tftypes.NewValue(rawType, map[string]tftypes.Value{
			"type":      siblingVal,
			"site_name": tftypes.NewValue(tftypes.String, nil),
		}),
	}
}

// TestRequiredIf_ListSibling verifies that a list sibling triggers the
// requirement only when a known string element matches, and that a sibling
// that is neither a string nor a list never triggers it.
func TestRequiredIf_ListSibling(t *testing.T) {
	strList := tftypes.List{ElementType: tftypes.String}
	intList := tftypes.List{ElementType: tftypes.Number}
	strListAttr := schema.ListAttribute{ElementType: types.StringType, Optional: true}

	tests := []struct {
		name    string
		cfg     tfsdk.Config
		wantErr bool
	}{
		{
			name: "list_match",
			cfg: buildSiblingConfig(t, strListAttr, strList, tftypes.NewValue(strList, []tftypes.Value{
				tftypes.NewValue(tftypes.String, nil),
				tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
				tftypes.NewValue(tftypes.String, "JWT"),
			})),
			wantErr: true,
		},
		{
			name: "list_no_match",
			cfg: buildSiblingConfig(t, strListAttr, strList, tftypes.NewValue(strList, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "OTHER"),
			})),
		},
		{
			name: "list_null",
			cfg:  buildSiblingConfig(t, strListAttr, strList, tftypes.NewValue(strList, nil)),
		},
		{
			name: "list_of_non_strings",
			cfg: buildSiblingConfig(t, schema.ListAttribute{ElementType: types.Int64Type, Optional: true}, intList,
				tftypes.NewValue(intList, []tftypes.Value{tftypes.NewValue(tftypes.Number, 1)})),
		},
		{
			name: "bool_sibling",
			cfg:  buildSiblingConfig(t, schema.BoolAttribute{Optional: true}, tftypes.Bool, tftypes.NewValue(tftypes.Bool, true)),
		},
	}

	v := requiredifvalidator.RequiredIf("type", "JWT")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("site_name"),
				Config:      tt.cfg,
				ConfigValue: types.StringNull(),
			}, resp)

			if got := resp.Diagnostics.HasError(); got != tt.wantErr {
				t.Errorf("HasError() = %v, want %v: %v", got, tt.wantErr, resp.Diagnostics)
			}
		})
	}
}

// TestRequiredIf_MarkdownDescriptionList verifies that three or more values
// are joined with commas and a final "or".
func TestRequiredIf_MarkdownDescriptionList(t *testing.T) {
	v := requiredifvalidator.RequiredIf("type", "A", "B", "C")

	want := "This field is required when `type` is set to `A`, `B` or `C`."
	if got := v.MarkdownDescription(context.Background()); got != want {
		t.Errorf("MarkdownDescription() = %q, want %q", got, want)
	}
}
