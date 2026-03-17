# Severity Guide

## CRITICAL (90+ confidence)
- Security vulnerability (injection, traversal, leaked secrets)
- Nil pointer dereference (panic in production)
- Data loss risk

## HIGH (75+ confidence)
- Convention violation that causes bugs (wrong error handling)
- Missing error check on external call
- Race condition

## MEDIUM (50+ confidence)  
- Convention violation (style, naming)
- Missing test for new code
- Unnecessary complexity

## LOW (25-49 confidence)
- Nitpick (could be intentional)
- Minor style inconsistency
- Suggestion for improvement
