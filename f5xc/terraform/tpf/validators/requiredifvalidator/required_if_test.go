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
