// Copyright Bunkerity 2025, 2026
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

// firstToken returns the first whitespace-separated token of a server_name,
// matching the BunkerWeb API's service identifier rule (server_name.split(" ")[0]).
// Unlike deriveServiceIdentifier it performs no case folding or sanitisation.
func firstToken(serverName string) string {
	fields := strings.Fields(serverName)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// isAffirmative reports whether a BunkerWeb boolean-ish setting value is true.
func isAffirmative(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "yes", "true", "on", "1":
		return true
	default:
		return false
	}
}

// lookupServiceSetting fetches a setting from a GET /services/{id} config map.
// The map is returned with unprefixed keys (verified against
// Database.get_non_default_settings, which strips the "{id}_" prefix for a single
// service); the prefixed form is tolerated defensively.
func lookupServiceSetting(cfg map[string]string, id, key string) (string, bool) {
	if v, ok := cfg[key]; ok {
		return v, true
	}
	if id != "" {
		if v, ok := cfg[id+"_"+key]; ok {
			return v, true
		}
	}
	return "", false
}

// serviceFromConfig reconstructs a service from a GET /services/{id} response for
// the read-only data source and ephemeral resource (which surface the full
// non-default settings set). The managed resource deliberately does NOT use this
// to avoid importing inherited settings as drift.
func serviceFromConfig(id string, cfg map[string]string) *bunkerWebService {
	svc := &bunkerWebService{ID: id, Variables: map[string]string{}}
	for k, v := range cfg {
		key := strings.TrimPrefix(k, id+"_")
		switch key {
		case "SERVER_NAME":
			svc.ServerName = v
		case "IS_DRAFT":
			svc.IsDraft = isAffirmative(v)
		default:
			svc.Variables[key] = v
		}
	}
	if svc.ServerName == "" {
		svc.ServerName = id
	}
	return svc
}
