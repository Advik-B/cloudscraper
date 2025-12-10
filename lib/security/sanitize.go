package security

import "strings"

// SanitizeDomain ensures the domain string is safe to use in JavaScript context
// by escaping special characters that could break out of string literals.
func SanitizeDomain(domain string) string {
// Replace backslash first to prevent double-escaping
domain = strings.ReplaceAll(domain, `\`, `\\`)
// Escape single quotes to prevent breaking out of string literals
domain = strings.ReplaceAll(domain, `'`, `\'`)
// Escape double quotes to prevent breaking out of double-quoted string literals
domain = strings.ReplaceAll(domain, `"`, `\"`)
// Escape newlines and other control characters
domain = strings.ReplaceAll(domain, "\n", `\n`)
domain = strings.ReplaceAll(domain, "\r", `\r`)
domain = strings.ReplaceAll(domain, "\t", `\t`)
return domain
}

// SanitizeDomainForJS ensures the domain string is safe to use in JavaScript context
// by filtering out potentially dangerous characters using a strict whitelist approach.
// Only alphanumeric characters, dots, hyphens, and colons (for port numbers) are allowed.
// Note: This may filter out underscores and internationalized domain names (IDN).
// For most Cloudflare challenges, standard ASCII domain names are expected.
func SanitizeDomainForJS(domain string) string {
// Only allow alphanumeric, dots, hyphens, and colons (for port numbers)
// This is a strict whitelist to prevent injection attacks
var result strings.Builder
for _, char := range domain {
if (char >= 'a' && char <= 'z') ||
(char >= 'A' && char <= 'Z') ||
(char >= '0' && char <= '9') ||
char == '.' || char == '-' || char == ':' {
result.WriteRune(char)
}
}
return result.String()
}
