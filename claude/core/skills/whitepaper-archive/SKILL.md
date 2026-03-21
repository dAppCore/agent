---
name: whitepaper-archive
description: Preserve whitepapers, technical documentation, and foundational documents from crypto projects
---

# Whitepaper Archive Collector

Preserve whitepapers, technical documentation, and foundational documents from crypto projects.

## Data Available

| Data Type | Source | Notes |
|-----------|--------|-------|
| Original whitepaper | Project site | PDF/HTML |
| Technical docs | GitHub wiki | Architecture details |
| Protocol specs | Docs site | Often disappear |
| Academic papers | arxiv, iacr | CryptoNote foundations |

## Known Sources

### CryptoNote Foundation
- Original CryptoNote whitepaper (van Saberhagen)
- Ring signature paper
- Stealth address paper

### Per-Project
- Monero Research Lab papers
- Haven Protocol whitepaper
- Lethean whitepaper

### Academic
- arxiv.org crypto papers
- iacr.org cryptography

## Usage

```bash
# Collect known whitepapers for a project
./generate-jobs.sh lethean > jobs.txt

# All CryptoNote foundational papers
./generate-jobs.sh --foundation > jobs.txt

# Research papers by topic
./generate-jobs.sh --topic=ring-signatures > jobs.txt
```

## Output

```
whitepapers/
├── cryptonote/
│   ├── cryptonote-v2.pdf
│   ├── ring-signatures.pdf
│   └── stealth-addresses.pdf
├── lethean/
│   ├── whitepaper-v1.pdf
│   └── technical-overview.md
└── INDEX.md
```

## Job Format

```
URL|FILENAME|TYPE|METADATA
https://cryptonote.org/whitepaper.pdf|cryptonote-v2.pdf|whitepaper|project=cryptonote,version=2
```

## Known URLs

### CryptoNote Original
- https://cryptonote.org/whitepaper.pdf (may be down)
- Archive.org backup needed

### Monero Research Lab
- https://www.getmonero.org/resources/research-lab/

### Academic
- https://eprint.iacr.org/ (IACR ePrint)
- https://arxiv.org/list/cs.CR/recent

## Notes

- Many original sites are gone - use Wayback Machine
- PDFs should be archived with multiple checksums
- Track version history when multiple revisions exist
