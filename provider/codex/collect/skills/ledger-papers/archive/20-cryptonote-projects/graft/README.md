# GraftNetwork Technical Documents

**Status:** Dead (2020)
**Salvage Priority:** HIGH
**Source:** github.com/graft-project/graft-ng

GraftNetwork was a CryptoNote-based payment network with supernode architecture for real-time authorization (RTA). The project died during crypto winter but left excellent technical documentation.

## Documents

| File | Original | Description |
|------|----------|-------------|
| RFC-001-GSD-general-supernode-design.md | Issue #187 | Supernode architecture, announce mechanism, key management |
| RFC-002-SLS-supernode-list-selection.md | Issue #185 | Auth sample selection algorithm |
| RFC-003-RTVF-rta-transaction-validation.md | Issue #191 | RTA validation flow + jagerman's security critique |
| auth-sample-selection-algorithm.md | Issue #182 | Randomness + stake weighting for sample selection |
| udht-implementation.md | Issue #341 | Unstructured DHT for supernode discovery |
| rta-double-spend-attack-vectors.md | Issue #425 | Attack matrix and solutions |
| RFC-005-DF-disqualification-flow.md | DesignDocs #2 | Disqualification scoring + jagerman critique |
| communication-options-p2p-design.md | DesignDocs #1 | 5 P2P architecture options with tradeoffs |
| blockchain-based-list-selection-analysis.md | GraftNetwork PR-225 | jagerman's 10M simulation statistical analysis |

## Key Insights

### From RFC 001 (jagerman's critique)
- Announce mechanism creates 60-144 GB/day network traffic
- Hop count in announcements leaks IP (not anonymous)
- Suggested fix: disqualification tx on-chain instead of gossip

### From RFC 003 (privacy analysis)
- Proxy SN sees: recipient wallet, amount, item list
- Auth sample sees: total amount
- Single point of failure in proxy design
- Solution: end-to-end encryption, zero-knowledge proofs

### From Attack Vectors
- RTA vs non-RTA: prioritize RTA, rollback conflicting blocks
- RTA vs RTA: shouldn't happen if auth sample honest
- Needs checkpoint depth limit

## Relevance to Lethean

- Service node architecture → Exit node incentives
- RTA validation → Session authorization
- Disqualification flow → Node quality enforcement
- UDHT → Decentralized service discovery
