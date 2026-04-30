---
name: security
description: Security-focused code review
args: [commit-range|--pr=N]
---

# Security Review

Perform a security-focused code review.

## Focus Areas

### 1. Injection Vulnerabilities
- SQL injection
- Command injection
- XSS (Cross-Site Scripting)
- LDAP injection
- XML injection

### 2. Authentication & Authorisation
- Hardcoded credentials
- Weak password handling
- Missing auth checks
- Privilege escalation paths

### 3. Data Exposure
- Sensitive data in logs
- PII in error messages
- Secrets in version control
- Insecure data transmission

### 4. Cryptography
- Weak algorithms (MD5, SHA1 for security)
- Hardcoded keys/IVs
- Insecure random number generation

### 5. Dependencies
- Known vulnerable packages
- Outdated dependencies

## Process

1. Get diff for specified range
2. Scan for security patterns
3. Check for common vulnerabilities
4. Report findings with severity

## Patterns to Check

### Go
```go
// SQL injection
db.Query("SELECT * FROM users WHERE id = " + id)

// Command injection
exec.Command("bash", "-c", userInput)

// Hardcoded secrets
apiKey := "sk_live_..."
```

### PHP
```php
// SQL injection
$db->query("SELECT * FROM users WHERE id = $id");

// XSS
echo $request->input('name');

// Command injection
shell_exec($userInput);
```

## Output Format

```markdown
## Security Review

### Critical
- **file:line** - SQL Injection: User input directly in query

### High
- **file:line** - Hardcoded API key detected

### Medium
- **file:line** - Missing CSRF protection

### Low
- **file:line** - Debug endpoint exposed

---
**Summary**: X critical, Y high, Z medium, W low
```
