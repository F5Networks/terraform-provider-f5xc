//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

// Package requiredifvalidator provides conditional Terraform Plugin Framework
// validators and plan modifiers for attributes whose requirement depends on
// sibling or root resource attributes.
//
// Why: Several F5 XC schema fields are optional in general but become mandatory
// when a sibling attribute (e.g. token "type" = "JWT") or a combination of
// sibling and root namespace attributes (e.g. referencing an object in the
// "shared" namespace from a non-shared resource namespace) is configured.
//
// What it provides:
//   - RequiredIfValidator / RequiredIf: validates that a string attribute is
//     set when a sibling string attribute matches one of the trigger values.
//   - CrossNamespaceSharedTenantValidator / RequireTenantForCrossNamespaceShared:
//     implements both validator.String and planmodifier.String to enforce at
//     plan time that ObjectRefType.tenant is configured whenever a resource in
//     a non-shared namespace references an object in the "shared" namespace.
package requiredifvalidator
