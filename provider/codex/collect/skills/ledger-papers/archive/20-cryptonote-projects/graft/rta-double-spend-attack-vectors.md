# Issue #425: Graft RTA Double Spend Attack Vectors and Solutions

## Reception Score

| Score | Reason |
|-------|--------|
| **STALE** | Open with no response |

---

## Metadata

| Field | Value |
|-------|-------|
| State | OPEN |
| Author | @mbg033 |
| Created | 2020-05-28 |
| Closed | N/A |
| Labels |  |
| Comments | 0 |

---

## Original Post

**Author:** @mbg033

| **&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Attack&nbsp;Vector&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;** | **How to implement attack** | **Solution** | **Comments/Questions** |
| --- | --- | --- | --- |
|  **Double Spend with Non-RTA TX (RTA vs non-RTA), classic 51% attack, below is the attack at the different states)** | **two possible scenarios addressed by [Jason](https://graftnetwork.atlassian.net/browse/SUP-51)** |  |  |
|  1. RTA vs non-RTA tx in mempool |  | Prioritize RTA over PoW. Conflicting non-RTA tx should be removed from pool as soon as RTA tx has been added; |  |
|  2. RTA tx in mempool vs non-RTA tx in mainchain | Longer chain with double spending TX published to the network right after someone completed RTA TX (signed RTA TX just added to mempool on some node and broadcased to the network) | Rollback: all blocks starting from block containing conflicting TX should be popped from blockchain, returning valid transactions to mempool, conflicting non-RTA transactions removed from mempool | Rollback should be (?) limited by the depth. In case checkpointing implemented - till first checkpoited (unreversible) block; if no checkpointing - N blocks max. N should be some reasonable constant |
|  3. RTA tx in mempool vs non-RTA tx in altchain |  | Rollback in alt chain if applicable | Question: check if rollbacks are applicable for alt chains, how it implemented |
|  4. RTA txs in mainchain vs non-RTA txes in altchains |  | Rollback (alt chain becames mainchain) until unreversible checkpoint or max possible depth (N) reached |  |
|  **Double Spend with RTA tx (RTA vs RTA)** | **Can't see how it possible - it needs to be maliciouls auth sample coexisting with true auth sample** |  |  |
|  1. RTA tx in mempool vs RTA tx in mainchain |  | in theory this shouldn't be possible: auth sample supernodes are checking for conflicting key images so such tx will never added to a pool. Only if malicious tx was accepted by malicious auth sample somehow | Question: check if it (how it) possible so we have more than one "valid" auth sample (i.e. one for main chain, another one(s) for alt chain(s), if main chain for one specific node is alt chain for another node |
|  2. RTA txs in mainchain vs RTA txes in altchain |  | in theory this shouldn't be possible: auth sample supernodes are checking for conflicting key images so such tx will never added to a pool. Only if malicious tx was accepted by malicious auth sample somehow |  |

---

## Discussion Thread

