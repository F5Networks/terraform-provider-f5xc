//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

// Package namespacetypevalidator validates namespace names against F5 XC namespace categories.
//
// The schema's in_namespace_types option describes categories such as system,
// shared, and user. User namespaces are arbitrary names, so this package
// mirrors backend category rules without treating category labels as names.
package namespacetypevalidator
