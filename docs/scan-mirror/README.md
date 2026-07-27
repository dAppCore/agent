<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Scan & mirror — the Forge ↔ GitHub seam

core/agent's tracker of record is **Forge**; GitHub is downstream. These tools bridge the
two and surface work.

| Tool / verb | What it does |
|-------------|--------------|
| `agentic_scan` | scan **Forge** issues — surface tracked work to [dispatch](../dispatch/) against |
| `agentic_mirror` | mirror **Forge → GitHub** (push the canonical Forge state downstream) |
| `agentic:repo/sync` (`repo/sync`) | freshen a single repo's working tree before a dispatch |

`agentic_scan` is the front door of the dispatch loop (find the issue → dispatch it);
`agentic_mirror` keeps GitHub a faithful downstream copy of Forge. QA findings ingested by
the [pipeline](../pipeline/) (`auto-ingest`) become Forge issues that `agentic_scan` then
picks up — closing the loop.

## Next

[dispatch](../dispatch/) (consumes `agentic_scan`) · [pipeline](../pipeline/) (produces
ingested findings) · [fleet](../fleet/) (`repo/sync` keeps fleet trees fresh).
