# SPDX-License-Identifier: EUPL-1.2

from __future__ import annotations

from contextlib import nullcontext
from unittest.mock import patch

from hermes.plugins.openbrain_context import OpenBrainContextEngine


def make_engine() -> OpenBrainContextEngine:
    return OpenBrainContextEngine(
        brain_url="https://brain.example",
        api_key="test-key",
        qdrant_url="https://qdrant.example",
        pg_dsn="postgresql://brain:secret@postgres.example:5432/openbrain",
        workspace_id=74,
        org="lthn",
    )


def make_turns() -> list[dict]:
    return [
        {
            "id": "turn-0",
            "role": "system",
            "content": "System context for Hermes and safety rules across the workspace.",
        },
        {
            "id": "turn-1",
            "role": "assistant",
            "content": "Old chatter about office snacks, coffee orders, and travel timings.",
        },
        {
            "id": "turn-2",
            "role": "assistant",
            "content": "OpenBrain qdrant recall centrality retrieval graph ranking memory compression context.",
        },
        {
            "id": "turn-3",
            "role": "user",
            "content": "Another diversion about keyboard colours, umbrellas, and station weather.",
        },
        {
            "id": "turn-4",
            "role": "assistant",
            "content": "Context compression should keep qdrant recall centrality retrieval graph ranking context.",
        },
        {
            "id": "turn-5",
            "role": "user",
            "content": "Please implement Hermes context compression with qdrant recall centrality retrieval.",
        },
    ]


def recall_payload() -> dict:
    return {
        "status": 200,
        "data": {
            "memories": [
                {
                    "id": "mem-1",
                    "content": "qdrant recall centrality retrieval graph ranking context compression",
                    "score": 0.97,
                },
                {
                    "id": "mem-2",
                    "content": "openbrain qdrant recall centrality retrieval graph ranking memory compression context",
                    "score": 0.95,
                },
                {
                    "id": "mem-3",
                    "content": "garden picnic sandwiches clouds trains umbrellas",
                    "score": 0.10,
                },
            ]
        },
    }


def test_is_available_happy() -> None:
    engine = make_engine()

    with patch.object(engine, "_request_status", return_value=200), patch(
        "hermes.plugins.openbrain_context.socket.create_connection",
        return_value=nullcontext(object()),
    ):
        assert engine.is_available() is True


def test_is_available_qdrant_down() -> None:
    engine = make_engine()

    with patch.object(engine, "_request_status", side_effect=OSError("down")), patch(
        "hermes.plugins.openbrain_context.socket.create_connection",
        return_value=nullcontext(object()),
    ):
        assert engine.is_available() is False


def test_compress_with_short_input_returns_turns_unchanged() -> None:
    engine = make_engine()
    turns = [
        {"role": "system", "content": "System prompt"},
        {"role": "user", "content": "Current request"},
    ]

    compressed = engine.compress(turns, token_budget=1)

    assert compressed == turns


def test_compress_with_recall_candidates_keeps_central_turns() -> None:
    engine = make_engine()
    turns = make_turns()

    with patch.object(engine, "_request_json", return_value=recall_payload()):
        compressed = engine.compress(turns, token_budget=80, top_k=2, candidate_pool=10)

    assert [turn["id"] for turn in compressed] == ["turn-0", "turn-2", "turn-4", "turn-5"]


def test_compress_preserves_first_and_last_turn_always() -> None:
    engine = make_engine()
    turns = make_turns()

    with patch.object(engine, "_request_json", return_value=recall_payload()):
        compressed = engine.compress(turns, token_budget=36, top_k=1, candidate_pool=10)

    assert compressed[0] == turns[0]
    assert compressed[-1] == turns[-1]
    assert turns[0] in compressed
    assert turns[-1] in compressed


def test_compress_falls_back_gracefully_when_recall_fails(capsys) -> None:
    engine = make_engine()
    turns = make_turns()

    with patch.object(engine, "_request_json", side_effect=OSError("down")):
        compressed = engine.compress(turns, token_budget=59, candidate_pool=10)

    assert [turn["id"] for turn in compressed] == ["turn-0", "turn-4", "turn-5"]
    assert "falling back to head+tail truncation" in capsys.readouterr().err
