// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import "testing"

func TestDeriveServiceIdentifier(t *testing.T) {
	cases := map[string]string{
		"My Service":          "my",
		"my-service":          "my-service",
		" my_service ":        "my-service",
		"":                    "service",
		"   ":                 "service",
		"Example.Com":         "example.com",
		"Example.Com extra":   "example.com",
		"UPSTREAM01":          "upstream01",
		"invalid chars !@#$%": "invalid",
	}

	for input, expected := range cases {
		actual := deriveServiceIdentifier(input)
		if actual != expected {
			t.Fatalf("deriveServiceIdentifier(%q) = %q, want %q", input, actual, expected)
		}
	}
}
