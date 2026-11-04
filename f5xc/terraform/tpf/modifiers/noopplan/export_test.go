//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

package noopplan

// ApplyWithPrivate exposes apply to external tests, which cannot build framework private state.
var ApplyWithPrivate = apply
