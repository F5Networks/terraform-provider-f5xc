//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package resource

import (
	"fmt"
	"strings"
	"sync"
)

var attachmentTargetLocks sync.Map

// lockAttachmentTarget serializes attachment read-modify-write operations for one parent resource.
func lockAttachmentTarget(resourceType, namespace, name string) func() {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = "default"
	}
	key := fmt.Sprintf("%s/%s/%s", resourceType, ns, strings.TrimSpace(name))

	muAny, _ := attachmentTargetLocks.LoadOrStore(key, &sync.Mutex{})
	mu := muAny.(*sync.Mutex)
	mu.Lock()

	return mu.Unlock
}
