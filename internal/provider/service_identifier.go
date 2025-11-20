// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import "strings"

// deriveServiceIdentifier normalizes a BunkerWeb service name into an identifier.
func deriveServiceIdentifier(serverName string) string {
	trimmed := strings.TrimSpace(serverName)
	if trimmed == "" {
		return "service"
	}

	parts := strings.Fields(trimmed)
	identifier := strings.ToLower(parts[0])

	var b strings.Builder
	b.Grow(len(identifier))

	for _, r := range identifier {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == '-':
			b.WriteRune(r)
		case r == '_' || r == ' ':
			b.WriteRune('-')
		}
	}

	result := b.String()
	if result == "" {
		return "service"
	}

	return result
}
