//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

// Package hook contains hand-written lifecycle hooks that generated F5XC Terraform
// resources call at specific points in their CRUD and import lifecycle.
//
// Why: some backend behavior cannot be expressed by the omnigen-generated resource
// code alone (for example, action-based credential renewal, vK8s namespace routing,
// or unbinding referring objects before delete). Hooks keep that logic in one
// manually maintained package instead of patching generated files.
//
// What it provides:
//   - Lifecycle hooks (pre/post create, update, delete) and resource state-change hooks.
//   - Override hooks that replace a generated operation (create, read, update, delete, import).
//   - Helpers used in generated call arguments, such as
//     ResolveApiCredentialNamespace and ResolveServiceCredentialNamespace.
//
// Hook call sites, arguments, and the {ResourcePascalCase}{Position}Hook naming
// convention are configured in omnigen (pkg/config.json "hooks"); the functions in
// this package are NOT auto-generated.
package hook
