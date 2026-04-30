# Ledger Papers Archive

Comprehensive collection of distributed ledger, cryptographic protocol, and decentralized systems whitepapers.

**For the commons - EUPL-1.2 CIC**

## Stats

- **91+ papers** across **15 categories**
- Genesis to modern (1998-2024)
- Academic + project whitepapers

## Categories

| Category | Papers | Description |
|----------|--------|-------------|
| genesis | 4 | Pre-Bitcoin: b-money, hashcash, bit gold |
| cryptonote | 2 | CryptoNote v2.0 + standards (CNS001-010) |
| mrl | 11 | Monero Research Lab (MRL-0001 to MRL-0011) |
| privacy | 9 | Zcash, Dash, Mimblewimble, Lelantus, Spark |
| smart-contracts | 10 | Ethereum, Solana, Cardano, Polkadot, etc |
| layer2 | 7 | Lightning, Plasma, Rollups, zkSync |
| consensus | 7 | PBFT, Tendermint, HotStuff, Casper |
| cryptography | 10 | Bulletproofs, CLSAG, PLONK, Schnorr, BLS |
| defi | 7 | Uniswap, Aave, Compound, Curve, MakerDAO |
| storage | 5 | IPFS, Filecoin, Arweave, Sia |
| identity | 3 | DIDs, Verifiable Credentials, Semaphore |
| cryptonote-projects | 5 | Haven, Masari, TurtleCoin, Wownero, DERO |
| attacks | 5 | Selfish mining, eclipse, traceability |
| oracles | 3 | Chainlink, Band Protocol |
| bridges | 3 | Atomic swaps, XCLAIM, THORChain |

## Collection with Borg

```bash
# Collect papers from academic sources
borg collect website https://eprint.iacr.org --depth 2 --format stim -o iacr-papers.stim

# Collect from arXiv
borg collect website https://arxiv.org/list/cs.CR/recent --depth 1

# Package existing archive
borg compile -f Borgfile -e "archive-password" -o ledger-papers.stim
```

## Registry

Papers are catalogued in `registry.json`:

```json
{
  "id": "paper-id",
  "title": "Paper Title",
  "year": 2024,
  "url": "https://example.com/paper.pdf",
  "topics": ["topic1", "topic2"]
}
```

## CDN Hosting Structure

```
papers.lethean.io/
├── genesis/
│   ├── bitcoin.pdf
│   ├── b-money.txt
│   └── hashcash.pdf
├── cryptonote/
│   ├── cryptonote-v2.pdf
│   └── cns/
├── mrl/
│   └── MRL-0001.pdf
├── cryptography/
│   ├── bulletproofs.pdf
│   └── clsag.pdf
└── INDEX.json
```

## License Note

Papers collected for archival/educational purposes. Original copyrights remain with authors.
