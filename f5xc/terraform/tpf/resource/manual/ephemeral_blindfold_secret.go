//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package manual

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	ephemeralschema "github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/utils/blindfold"
)

// blindfoldSecretModel maps the ephemeral resource schema to a Go type. The
// content attribute is marked write-only + sensitive so that Terraform never
// persists the raw plaintext to .tfstate. Only the resulting `location`
// ciphertext (string:///...) is returned to the practitioner.
type blindfoldSecretModel struct {
	Namespace  types.String `tfsdk:"namespace"`
	PolicyName types.String `tfsdk:"policy_name"`
	Content    types.String `tfsdk:"content"`
	KeyVersion types.Int64  `tfsdk:"key_version"`
	Location   types.String `tfsdk:"location"`
	PolicyID   types.Int64  `tfsdk:"policy_id"`
	Tenant     types.String `tfsdk:"tenant"`
}

// EphemeralF5XCBlindfoldSecret implements ephemeral "f5xc_blindfold_secret".
type EphemeralF5XCBlindfoldSecret struct {
	client *client.APIClient
}

var (
	_ ephemeral.EphemeralResource              = &EphemeralF5XCBlindfoldSecret{}
	_ ephemeral.EphemeralResourceWithConfigure = &EphemeralF5XCBlindfoldSecret{}
)

// NewEphemeralF5XCBlindfoldSecret is the ephemeral resource factory registered
// with the provider.
func NewEphemeralF5XCBlindfoldSecret() ephemeral.EphemeralResource {
	return &EphemeralF5XCBlindfoldSecret{}
}

func (r *EphemeralF5XCBlindfoldSecret) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blindfold_secret"
}

func (r *EphemeralF5XCBlindfoldSecret) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = ephemeralschema.Schema{
		Description: "Client-side Blindfold envelope-encrypts a plaintext secret using the tenant public key " +
			"and a bound secret policy. The raw content is only held in memory during terraform apply and is " +
			"never persisted to .tfstate. Only the resulting string:/// ciphertext URL is exported.",
		MarkdownDescription: "Client-side Blindfold envelope-encrypts a plaintext secret using the tenant public " +
			"key and a bound secret policy. The raw `content` is only held in memory during `terraform apply` " +
			"and is never persisted to `.tfstate`. Only the resulting `string:///` ciphertext URL is exported.",
		Attributes: map[string]ephemeralschema.Attribute{
			"namespace": ephemeralschema.StringAttribute{
				Required:            true,
				Description:         "Namespace of the F5XC Secret Policy to bind the ciphertext to.",
				MarkdownDescription: "Namespace of the F5XC Secret Policy to bind the ciphertext to.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"policy_name": ephemeralschema.StringAttribute{
				Required:            true,
				Description:         "Name of the F5XC Secret Policy that authorizes runtime decryption.",
				MarkdownDescription: "Name of the F5XC Secret Policy that authorizes runtime decryption.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"content": ephemeralschema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				Description:         "Plaintext secret payload to blindfold. Held only in memory during apply.",
				MarkdownDescription: "Plaintext secret payload to blindfold. Held only in memory during apply.",
			},
			"key_version": ephemeralschema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Optional public key version. Defaults to the latest tenant key when unset.",
				MarkdownDescription: "Optional public key version. Defaults to the latest tenant key when unset.",
			},
			"location": ephemeralschema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				Description:         "The blindfolded ciphertext URL in string:///<base64> form.",
				MarkdownDescription: "The blindfolded ciphertext URL in `string:///<base64>` form.",
			},
			"policy_id": ephemeralschema.Int64Attribute{
				Computed:            true,
				Description:         "The numeric F5XC policy identifier that was bound to the ciphertext header.",
				MarkdownDescription: "The numeric F5XC policy identifier that was bound to the ciphertext header.",
			},
			"tenant": ephemeralschema.StringAttribute{
				Computed:            true,
				Description:         "The tenant returned by the F5XC secret-policy document.",
				MarkdownDescription: "The tenant returned by the F5XC secret-policy document.",
			},
		},
	}
}

func (r *EphemeralF5XCBlindfoldSecret) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	apiClient, ok := req.ProviderData.(*client.APIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Ephemeral Resource Configure Type",
			fmt.Sprintf("Expected *client.APIClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = apiClient
}

func (r *EphemeralF5XCBlindfoldSecret) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data blindfoldSecretModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil || r.client.RestClient == nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			"The F5XC provider REST client is not configured. Ensure the provider block sets valid API credentials before using ephemeral.f5xc_blindfold_secret.",
		)
		return
	}

	if data.Content.IsNull() || data.Content.IsUnknown() || data.Content.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("content"),
			"Empty content",
			"content must be a non-empty plaintext value to blindfold.",
		)
		return
	}

	engine := &blindfold.Client{
		Exec:    r.client.RestClient,
		BaseURL: r.client.RestClient.URL,
	}

	keyData, err := engine.GetPublicKey(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Blindfold: fetch public key failed", err.Error())
		return
	}
	if !data.KeyVersion.IsNull() && !data.KeyVersion.IsUnknown() {
		keyData.KeyVersion = uint32(data.KeyVersion.ValueInt64())
	}

	policyData, err := engine.GetPolicyDocument(ctx, data.Namespace.ValueString(), data.PolicyName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Blindfold: fetch policy document failed", err.Error())
		return
	}

	location, err := blindfold.Blindfold([]byte(data.Content.ValueString()), keyData, policyData)
	if err != nil {
		resp.Diagnostics.AddError("Blindfold: envelope encryption failed", err.Error())
		return
	}

	tflog.Info(ctx, "Blindfold: envelope encryption succeeded", map[string]any{
		"namespace":   policyData.Namespace,
		"policy_name": policyData.Name,
		"policy_id":   policyData.PolicyID,
		"tenant":      policyData.Tenant,
	})

	data.Location = types.StringValue(location)
	data.PolicyID = types.Int64Value(int64(policyData.PolicyID))
	data.Tenant = types.StringValue(policyData.Tenant)
	if data.KeyVersion.IsNull() || data.KeyVersion.IsUnknown() {
		data.KeyVersion = types.Int64Value(int64(keyData.KeyVersion))
	}

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}
