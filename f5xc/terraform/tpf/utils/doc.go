// Copyright (c) 2025 F5 Inc. All rights reserved.

// Package utils provides utility functions for the Terraform Provider Framework (TPF).
//
// The utils package centralizes common conversion, error handling, and resource lifecycle patterns
// used throughout the provider implementation. It includes:
//   - Conversion functions between Go types and TPF types (ObjectValue, MapValue, ListValue)
//   - Default value helpers for schema definitions
//   - Time parsing utilities for API and state mapping
//   - Resource initializer polling and error handling
//   - Diagnostics propagation for framework compatibility
//
// These utilities are designed to reduce code duplication and ensure consistent error handling
// and type safety across all resources and data sources in the provider.
//
// Example usage:
//
//	import "gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/utils"
//
//	// Convert a Go struct slice to a TPF ListValue
//	listValue, diags := utils.GoStructSliceToListValue(ctx, attrMap, goSlice)
//
//	// Wait for pending initializers on a resource
//	err := utils.WaitForPendingInitializers(ctx, name, namespace, uid, getObjectFn)
//
// See individual function documentation for details.
package utils
