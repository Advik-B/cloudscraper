# Security Policy

## Overview

This document outlines the security measures implemented in the cloudscraper library to protect against Remote Code Execution (RCE) and other security vulnerabilities.

## Security Measures

### 1. Domain Sanitization

#### v1 Challenge Protection
- **File**: `lib/challenge_v1.go`
- **Function**: `sanitizeDomain()`
- **Protection**: Escapes special characters in domain strings to prevent JavaScript injection
- **Implementation**: 
  - Escapes backslashes (`\` → `\\`)
  - Escapes single quotes (`'` → `\'`)
  - Escapes double quotes (`"` → `\"`)
  - Escapes control characters (newlines, tabs, carriage returns)

#### v2 Challenge Protection
- **File**: `lib/challenge_v2.go`
- **Function**: `sanitizeDomainForJS()`
- **Protection**: Uses strict whitelist filtering to prevent injection attacks
- **Implementation**: Only allows alphanumeric characters, dots (`.`), hyphens (`-`), and colons (`:`)

### 2. Script Size Limits

#### External Engine Protection
- **File**: `lib/js/external_engine.go`
- **Limit**: 5 MB (5,242,880 bytes)
- **Protection**: Prevents Denial of Service (DoS) attacks through excessively large scripts
- **Error**: Returns error if script exceeds maximum size

#### Goja Engine Protection
- **File**: `lib/js/goja_engine.go`
- **Limit**: 5 MB (5,242,880 bytes)
- **Protection**: Prevents DoS attacks and memory exhaustion
- **Error**: Returns error if script exceeds maximum size

#### v2 Challenge Script Protection
- **File**: `lib/challenge_v2.go`
- **Limit**: 1 MB (1,048,576 bytes)
- **Protection**: Additional layer of protection for challenge scripts
- **Scope**: Applies to total size of all extracted challenge scripts

### 3. Command Injection Prevention

#### External Engine Validation
- **File**: `lib/js/external_engine.go`
- **Function**: `NewExternalEngine()`
- **Protection**: Only allows whitelisted JavaScript runtimes
- **Allowed Commands**: `node`, `deno`, `bun`
- **Implementation**: Uses switch statement to validate commands before execution

### 4. Execution Timeout Protection

#### Goja Engine Timeout
- **File**: `lib/js/goja_engine.go`
- **Function**: `Run()`
- **Timeout**: 3 seconds
- **Protection**: Prevents infinite loops and resource exhaustion
- **Implementation**: Uses goroutine with timeout and panic recovery

## Threat Model

### Mitigated Threats

1. **JavaScript Injection via Domain Names**
   - **Attack Vector**: Malicious domain names containing quotes or special characters
   - **Mitigation**: Domain sanitization functions
   - **Impact**: Prevents arbitrary JavaScript code execution

2. **Denial of Service via Large Scripts**
   - **Attack Vector**: Extremely large scripts causing memory exhaustion
   - **Mitigation**: Script size limits
   - **Impact**: Prevents service disruption

3. **Command Injection via Custom Runtime**
   - **Attack Vector**: Arbitrary command execution through custom JS runtime
   - **Mitigation**: Whitelist validation
   - **Impact**: Prevents system compromise

4. **Resource Exhaustion via Infinite Loops**
   - **Attack Vector**: Scripts with infinite loops or long-running operations
   - **Mitigation**: Execution timeouts
   - **Impact**: Prevents resource exhaustion

### Residual Risks

1. **Trusted Script Execution**
   - **Risk**: The library executes JavaScript from Cloudflare challenge pages
   - **Mitigation**: Uses sandboxed VM (Goja) with limited capabilities
   - **Assumption**: Cloudflare challenge pages are trusted
   - **Note**: This is an inherent risk of the library's functionality

2. **Custom JS Engine Implementation**
   - **Risk**: Users can provide custom JS engine via `WithCustomJSEngine()`
   - **Mitigation**: None - users are responsible for their custom implementations
   - **Recommendation**: Only use trusted custom engines

3. **Domain Name Limitations**
   - **Risk**: The v2 challenge domain sanitization may filter out valid internationalized domain names (IDN) and domains with underscores
   - **Impact**: Such domains may fail to solve challenges
   - **Mitigation**: This is a trade-off for security - most Cloudflare-protected sites use standard ASCII domain names
   - **Note**: If you encounter issues with non-standard domains, please report them

## Security Best Practices

### For Library Users

1. **Use Default Engine**: Prefer the built-in Goja engine over external runtimes
2. **Validate Domains**: Only scrape trusted domains
3. **Monitor Resource Usage**: Set appropriate timeouts and memory limits
4. **Keep Updated**: Regularly update to the latest version for security patches
5. **Avoid Custom Engines**: Only use custom JS engines from trusted sources

### For Contributors

1. **Input Validation**: Always validate and sanitize user-controlled input
2. **Size Limits**: Implement size limits for all external data
3. **Timeout Protection**: Use timeouts for all potentially long-running operations
4. **Whitelist Approach**: Prefer whitelisting over blacklisting for security checks
5. **Security Reviews**: All PRs should undergo security review

## Reporting Security Issues

If you discover a security vulnerability, please report it via:
- **GitHub Security Advisories**: Use the "Security" tab on the GitHub repository to create a private security advisory
- **Issue Tracker**: For non-critical security concerns, you can create a GitHub issue

Please do **NOT** create public issues for critical security vulnerabilities. Use GitHub's private security advisory feature instead.

## Security Changelog

### 2025-12-10
- Added domain sanitization for v1 and v2 challenge solvers
- Implemented script size limits (1MB for v2 challenges, 5MB for engines)
- Enhanced command injection protection
- Added comprehensive security documentation
- CodeQL scan: 0 vulnerabilities found

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CWE-94: Code Injection](https://cwe.mitre.org/data/definitions/94.html)
- [CWE-78: Command Injection](https://cwe.mitre.org/data/definitions/78.html)
- [CWE-400: Resource Exhaustion](https://cwe.mitre.org/data/definitions/400.html)
