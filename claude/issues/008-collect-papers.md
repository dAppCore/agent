# feat(collect): Add research paper/whitepaper collection

## Summary

Add `core collect papers` command to discover and collect distributed ledger research papers and whitepapers.

## Required Commands

```bash
core collect papers --all                     # All known papers
core collect papers --category=cryptography   # Filter by category
core collect papers --topic=bulletproofs      # Filter by topic
core collect papers --project=monero          # Filter by project
core collect papers --search-iacr             # Search IACR eprint
core collect papers queue <url>               # Queue single paper
```

## Current Shell Scripts Being Replaced

- `claude/skills/ledger-papers/discover.sh` - Paper discovery
- `claude/collection/collect-whitepaper.sh` - Queue whitepaper collection
- `claude/collection/update-index.sh` - Update paper index

## Categories

- genesis (Bitcoin, b-money, hashcash, bit gold)
- cryptonote (CryptoNote v2.0, CNS standards)
- mrl (Monero Research Lab papers)
- privacy (Zcash, Dash, Mimblewimble, Lelantus)
- smart-contracts (Ethereum, Solana, Cardano)
- layer2 (Lightning, Plasma, Rollups)
- consensus (PBFT, Tendermint, HotStuff)
- cryptography (Bulletproofs, CLSAG, PLONK)
- defi (Uniswap, Aave, Compound)
- storage (IPFS, Filecoin, Arweave)
- identity (DIDs, Verifiable Credentials)
- attacks (Selfish mining, eclipse, traceability)

## Registry

Papers defined in `registry.json`:
```json
{
  "id": "bulletproofs",
  "title": "Bulletproofs: Short Proofs for Confidential Transactions",
  "year": 2017,
  "url": "https://eprint.iacr.org/2017/1066.pdf",
  "topics": ["range-proofs", "zero-knowledge"]
}
```

## Output Format

Job list for collection:
```
URL|FILENAME|TYPE|METADATA
https://bitcoin.org/bitcoin.pdf|bitcoin.pdf|paper|category=genesis,title=Bitcoin...
```

Summary:
```json
{
  "papers": 91,
  "categories": 15,
  "queued": 3,
  "output": "papers/"
}
```
