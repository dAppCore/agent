# Host UK Core Agent

This extension provides tools and workflows for the Host UK development environment.
It helps with code review, verification, QA, and CI tasks.

## Key Features

- **Core CLI Integration**: Enforces the use of `core` CLI (`host-uk/core` wrapper for go/php tools) to ensure consistency.
- **Auto-formatting**: Automatically formats Go and PHP code on edit.
- **Safety Checks**: Blocks destructive commands like `rm -rf` to prevent accidents.
- **Skills**: Provides data collection skills for various crypto/blockchain domains (e.g., Ledger papers, BitcoinTalk archives).
- **Codex Awareness**: Surfaces Codex guidance from `core-agent/codex/AGENTS.md`.
- **Ethics Modal**: Embeds the Axioms of Life ethics modal and strings safety guardrails.

## Codex Commands

- `/codex:awareness` - Show full Codex guidance.
- `/codex:overview` - Show Codex plugin overview.
- `/codex:core-cli` - Show core CLI mapping.
- `/codex:safety` - Show safety guardrails.
