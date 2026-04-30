---
name: security
description: Security-focused code review
args: [commit-range|--pr=N]
---

# Security Review

Perform a security-focused review of the requested changes.

## Focus Areas

1. Injection vulnerabilities
2. Authentication and authorisation
3. Data exposure
4. Cryptography and secret handling
5. Vulnerable or outdated dependencies

## Output

Return findings grouped by severity with file and line references, followed by a short summary count.
