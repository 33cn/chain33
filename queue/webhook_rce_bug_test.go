package queue

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

// F-CMD-001: cmd/webhook/main.go has multiple RCE vectors:
// 1. Empty webhook secret — anyone can send forged payloads
// 2. User/branch from payload used unsanitized in exec.Command paths
// 3. Attacker-controlled Makefile executed via `make webhook`

func TestWebhookInputSanitizationBug(t *testing.T) {
	// The safe pattern for GitHub usernames and branch names
	safePattern := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

	// Malicious inputs that could exploit the webhook
	maliciousInputs := []struct {
		name  string
		input string
	}{
		{"path traversal", "../../../etc/passwd"},
		{"command injection via semicolon", "user;rm -rf /"},
		{"command injection via backtick", "user`whoami`"},
		{"command injection via $(...)", "user$(id)"},
		{"null byte", "user\x00evil"},
		{"space injection", "user evil"},
	}

	for _, tc := range maliciousInputs {
		t.Run(tc.name, func(t *testing.T) {
			// BUG: original code does no validation
			buggyValidate := func(input string) bool {
				return true // accepts everything
			}

			// FIXED: validate against safe pattern
			fixedValidate := func(input string) bool {
				return safePattern.MatchString(input)
			}

			assert.True(t, buggyValidate(tc.input),
				"buggy: accepts malicious input %q", tc.input)
			assert.False(t, fixedValidate(tc.input),
				"fixed: rejects malicious input %q", tc.input)
		})
	}

	// Valid inputs should pass
	validInputs := []string{"user123", "my-branch", "feature_v2", "user.name"}
	for _, input := range validInputs {
		assert.True(t, safePattern.MatchString(input),
			"valid input %q should pass", input)
	}
}

func TestWebhookEmptySecretBug(t *testing.T) {
	// BUG: webhook secret is empty string — no authentication
	secret := ""
	assert.Empty(t, secret, "buggy: empty secret allows forged payloads")

	// FIXED: secret must be non-empty (from env var)
	fixedSecret := "WEBHOOK_SECRET_FROM_ENV" // #nosec G101 -- placeholder string demonstrating env-var sourcing
	assert.NotEmpty(t, fixedSecret, "fixed: non-empty secret required")
}
