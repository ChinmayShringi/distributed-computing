// Package redact provides secret redaction for logs and incident reports
package redact

import (
	"regexp"
	"strings"
)

var (
	// Common secret patterns
	patterns = []*regexp.Regexp{
		// API keys
		regexp.MustCompile(`(?i)(api[_-]?key|apikey)["\s:=]+["']?([a-zA-Z0-9_-]{20,})["']?`),
		// Tokens
		regexp.MustCompile(`(?i)(token|bearer)["\s:=]+["']?([a-zA-Z0-9_.-]{20,})["']?`),
		// Passwords
		regexp.MustCompile(`(?i)(password|passwd|pwd)["\s:=]+["']?([^"'\s]{4,})["']?`),
		// Secrets
		regexp.MustCompile(`(?i)(secret|private[_-]?key)["\s:=]+["']?([a-zA-Z0-9_/+=.-]{20,})["']?`),
		// AWS credentials
		regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		regexp.MustCompile(`(?i)aws[_-]?secret[_-]?access[_-]?key["\s:=]+["']?([a-zA-Z0-9/+=]{40})["']?`),
		// GitHub tokens
		regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),
		regexp.MustCompile(`gho_[a-zA-Z0-9]{36}`),
		regexp.MustCompile(`ghu_[a-zA-Z0-9]{36}`),
		// Database URLs with passwords
		regexp.MustCompile(`(?i)(postgres|mysql|mongodb)://[^:]+:([^@]+)@`),
		// JWT tokens
		regexp.MustCompile(`eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*`),
		// SSH private keys
		regexp.MustCompile(`-----BEGIN (RSA |DSA |EC |OPENSSH )?PRIVATE KEY-----`),
		// IP addresses (internal)
		regexp.MustCompile(`\b(10\.\d{1,3}\.\d{1,3}\.\d{1,3}|172\.(1[6-9]|2[0-9]|3[01])\.\d{1,3}\.\d{1,3}|192\.168\.\d{1,3}\.\d{1,3})\b`),
	}

	// Replacement text
	redactedText = "[REDACTED]"
)

// RedactSecrets redacts sensitive information from text
func RedactSecrets(text string) string {
	result := text

	for _, pattern := range patterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			// For patterns with capture groups, replace only the secret part
			if strings.Contains(match, "=") || strings.Contains(match, ":") {
				parts := regexp.MustCompile(`[=:]`).Split(match, 2)
				if len(parts) == 2 {
					return parts[0] + "=" + redactedText
				}
			}
			return redactedText
		})
	}

	return result
}

// RedactEnv redacts environment variables
func RedactEnv(env []string) []string {
	sensitiveKeys := []string{
		"PASSWORD", "SECRET", "TOKEN", "KEY", "CREDENTIAL",
		"AWS_", "GITHUB_", "API_", "AUTH_", "PRIVATE",
	}

	redacted := make([]string, len(env))
	for i, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			redacted[i] = e
			continue
		}

		key := strings.ToUpper(parts[0])
		isSensitive := false
		for _, sk := range sensitiveKeys {
			if strings.Contains(key, sk) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			redacted[i] = parts[0] + "=" + redactedText
		} else {
			redacted[i] = e
		}
	}

	return redacted
}

// RedactPath redacts sensitive path components
func RedactPath(path string) string {
	// Redact home directory username
	homePattern := regexp.MustCompile(`/Users/([^/]+)/`)
	path = homePattern.ReplaceAllString(path, "/Users/[USER]/")

	homePattern2 := regexp.MustCompile(`/home/([^/]+)/`)
	path = homePattern2.ReplaceAllString(path, "/home/[USER]/")

	return path
}

// ContainsSecret checks if text contains potential secrets
func ContainsSecret(text string) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(text) {
			return true
		}
	}
	return false
}
