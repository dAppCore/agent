---
name: debug
description: Systematic debugging workflow
---

# Debugging Protocol

## Step 1: Reproduce
- Run the failing test/command
- Note exact error message
- Identify conditions for failure

## Step 2: Isolate
- Binary search through changes (git bisect)
- Comment out code sections
- Add logging at key points

## Step 3: Hypothesize
Before changing code, form theories:
1. Theory A: ...
2. Theory B: ...

## Step 4: Test Hypotheses
Test each theory with minimal investigation.

## Step 5: Fix
Apply the smallest change that fixes the issue.

## Step 6: Verify
- Run original failing test
- Run full test suite
- Check for regressions
