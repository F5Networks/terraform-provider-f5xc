//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

// Package noopplan collapses update plans that contain no user-visible change back
// to the prior state.
//
// Why: Terraform core proposes null for Optional+Computed nested objects whose prior
// state holds non-computed children (e.g. a backend-defaulted oneof branch). The
// framework then treats every plan as a change and marks all unconfigured Computed
// attributes unknown, producing perpetual "(known after apply)" diffs for any
// Computed attribute that is not pinned by a plan modifier.
//
// What it provides:
//   - Apply: a resource-level ModifyPlan helper that replaces unknown values at
//     null-config paths with prior state and, if the result equals prior state,
//     makes the plan a no-op. Real changes keep their unknowns so backend-computed
//     values are always accepted on apply.
//   - RecordConfig: stores a deterministic config fingerprint in private state after
//     Create/Update so Apply can tell real config changes (including removal of an
//     Optional+Computed block) from spurious proposed changes.
package noopplan
