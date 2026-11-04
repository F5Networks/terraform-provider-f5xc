//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"regexp"
	"strings"
)

// ReferringResource represents a resource referenced by a policy.
type ReferringResource struct {
	Kind      string
	Namespace string
	Name      string
}

// ReferringHTTPLoadBalancer is an alias for ReferringResource for backward compatibility.
type ReferringHTTPLoadBalancer = ReferringResource

// ParseReferringObjects extracts referring resources from an F5XC API 409 error.
func ParseReferringObjects(errText string) []ReferringResource {
	var result []ReferringResource
	textToParse := errText
	if idx := strings.Index(errText, "is still being referred by"); idx != -1 {
		textToParse = errText[idx:]
	}

	re := regexp.MustCompile(`ves\.io\.schema\.([a-zA-Z0-9_\.]+)\.Object\s+([^\s\]]+)`)
	for _, match := range re.FindAllStringSubmatch(textToParse, -1) {
		if len(match) < 3 {
			continue
		}
		parts := strings.Split(match[2], "/")
		if len(parts) < 2 {
			continue
		}
		kindParts := strings.Split(match[1], ".")
		result = append(result, ReferringResource{
			Kind:      kindParts[len(kindParts)-1] + "s",
			Namespace: parts[len(parts)-2],
			Name:      parts[len(parts)-1],
		})
	}
	return result
}

// ParseReferringHTTPLoadBalancers is an alias for ParseReferringObjects for backward compatibility.
func ParseReferringHTTPLoadBalancers(errText string) []ReferringHTTPLoadBalancer {
	return ParseReferringObjects(errText)
}
