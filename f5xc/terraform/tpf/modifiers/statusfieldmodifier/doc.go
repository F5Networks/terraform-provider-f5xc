//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

// Package statusfieldmodifier provides Terraform plan modifiers for asynchronous,
// backend-managed status fields whose population depends on another configuration attribute.
//
// Why: In F5 XC resources (such as http_loadbalancer, tcp_loadbalancer, udp_loadbalancer),
// fields like host_name and dns_info are computed status attributes populated asynchronously
// by the backend depending on the configured advertise_choice. Standard modifiers either
// produce false drifts (when omitted) or cause "Provider produced inconsistent result after apply"
// errors (when UseStateForUnknown is used on null state or UseNonNullStateForUnknown is used on de-advertising).
//
// What it provides:
//   - UseStateUnlessDependencyChangesString: planmodifier.String implementation
//   - UseStateUnlessDependencyChangesList: planmodifier.List implementation
//   - UseStateUnlessDependencyChangesObject: planmodifier.Object implementation
//   - Automatic state retention when dependency is unchanged (no drift)
//   - Unknown plan preservation on null/unknown prior state (asynchronous initial population)
//   - Automatic transition to Unknown when controlling dependency changes (re-allocation / de-advertising)
package statusfieldmodifier
