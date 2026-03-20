# Python Build Flow

1. `python -m venv .venv && source .venv/bin/activate` — virtualenv
2. `pip install -e ".[dev]"` — install with dev deps
3. `ruff check .` — lint
4. `ruff format --check .` — format check
5. `pytest` — run tests
