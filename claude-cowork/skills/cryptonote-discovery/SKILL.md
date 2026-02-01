# CryptoNote Project Discovery

Discover and catalog CryptoNote-based projects for archival.

## Known CryptoNote Forks (2014-2024)

### Still Active
| Project | Symbol | Genesis | Status | Notable Features |
|---------|--------|---------|--------|------------------|
| Monero | XMR | 2014-04 | Active | RingCT, Bulletproofs |
| Haven | XHV | 2018-04 | Active | Synthetic assets |
| Wownero | WOW | 2018-04 | Active | Meme coin, RandomX |
| Dero | DERO | 2017-12 | Active | Smart contracts |
| Lethean | LTHN | 2017-10 | Active | dVPN/Proxy services |
| Karbo | KRB | 2016-05 | Active | Ukrainian community |

### Abandoned (Salvage Candidates)
| Project | Symbol | Genesis | Death | Reason | Salvageable |
|---------|--------|---------|-------|--------|-------------|
| Bytecoin | BCN | 2012-07 | 2022 | Premine scandal | Protocol research |
| Electroneum | ETN | 2017-09 | Pivot | Went mobile-only | Mobile wallet code |
| Aeon | AEON | 2014-06 | 2021 | Dev abandoned | Lightweight client |
| Masari | MSR | 2017-09 | 2022 | Dev MIA | Uncle mining |
| Loki | LOKI | 2018-03 | Rebrand | Now Session | Service nodes |
| Sumokoin | SUMO | 2017-04 | 2021 | Drama | Privacy features |
| Ryo | RYO | 2018-07 | 2023 | Low activity | GPU algo work |
| Conceal | CCX | 2018-01 | Low | Minimal dev | Banking features |
| Qwertycoin | QWC | 2018-01 | Low | Small team | Easy mining |
| TurtleCoin | TRTL | 2017-12 | 2023 | Team burnout | Community tools |
| Nerva | XNV | 2018-05 | 2022 | Solo mining only | Anti-pool algo |

## Data Sources Per Project

```
For each CryptoNote project, collect:

1. GitHub/GitLab repos
   - Core daemon
   - Wallet (CLI, GUI, mobile)
   - Pool software
   - Block explorer
   - Documentation

2. BitcoinTalk ANN thread
   - Original announcement
   - Updates
   - Community discussion

3. Block explorer
   - Genesis block
   - Emission curve
   - Network stats history

4. CoinMarketCap/CoinGecko
   - Price history
   - Description
   - Social links

5. Reddit/Discord
   - Archived discussions
   - Feature requests

6. Wayback Machine
   - Old website versions
   - Documentation snapshots
```

## Usage

```bash
# Discover all sources for a project
./discover.sh monero > monero-sources.txt
./discover.sh lethean > lethean-sources.txt

# Batch discover abandoned projects
./discover.sh --abandoned > salvage-targets.txt

# Generate collection jobs for all sources
./generate-all-jobs.sh lethean > lethean-jobs.txt
```

## Project Registry

The skill maintains a registry of known CryptoNote projects with:
- GitHub org/repos
- BitcoinTalk topic IDs
- Block explorer URLs
- CMC/CoinGecko slugs
- Social links
- Status (active/abandoned/dead)
- Notable innovations worth salvaging
