//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

// Package inlinereferencevalidator provides a Terraform object validator that prevents the use of
// restricted/unsafe inline reference choices at plan-time to avoid destroy ordering deadlocks (HTTP 400 RefCounter).
package inlinereferencevalidator

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// Ensure InlineReferenceValidator implements validator.Object interface.
var _ validator.Object = InlineReferenceValidator{}

// InlineReferenceValidator is a Terraform object validator that checks a choice/oneof attribute block
// and reports a plan-time diagnostic error if a forbidden choice attribute is configured.
type InlineReferenceValidator struct {
	ForbiddenChoices  []string
	SuggestedResource string
}

// RestrictedInlineReference returns a new InlineReferenceValidator configured with forbidden choices
// and a suggested attachment resource name.
func RestrictedInlineReference(forbiddenChoices []string, suggestedResource string) validator.Object {
	return InlineReferenceValidator{
		ForbiddenChoices:  forbiddenChoices,
		SuggestedResource: suggestedResource,
	}
}

// Description returns a human-readable description of the validator.
func (v InlineReferenceValidator) Description(_ context.Context) string {
	return fmt.Sprintf("Ensures inline choices %v are not configured, suggesting %s instead.", v.ForbiddenChoices, v.SuggestedResource)
}

// MarkdownDescription returns a markdown-formatted description of the validator.
func (v InlineReferenceValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateObject checks if any forbidden choice inside the target attribute block is set in the configuration.
func (v InlineReferenceValidator) ValidateObject(_ context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() {
		return
	}

	attrs := req.ConfigValue.Attributes()
	for _, forbidden := range v.ForbiddenChoices {
		forbiddenLower := strings.ToLower(forbidden)
		for attrKey, attrVal := range attrs {
			if strings.ToLower(attrKey) == forbiddenLower && !attrVal.IsNull() {
				resp.Diagnostics.AddAttributeError(
					req.Path,
					"Conflicting Service Policy Configuration",
					fmt.Sprintf(
						"Configuring '%s' in 'service_policy_choice' on HTTP LoadBalancer is incompatible with '%s'.\n"+
							"To manage service policies, please remove 'service_policy_choice' from the HTTP LoadBalancer resource and use '%s' instead.",
						attrKey, v.SuggestedResource, v.SuggestedResource,
					),
				)
				return
			}
		}
	}
}
