# SPDX-License-Identifier: EUPL-1.2

from __future__ import annotations

from contextlib import nullcontext
from unittest.mock import patch

from hermes.plugins.openbrain_memory import OpenBrainMemoryProvider


def make_provider() -> OpenBrainMemoryProvider:
    return OpenBrainMemoryProvider(
        brain_url="https://brain.example",
        api_key="test-key",
        qdrant_url="https://qdrant.example",
        pg_dsn="postgresql://brain:secret@postgres.example:5432/openbrain",
        workspace_id=73,
        org="lthn",
    )


def test_is_available_happy() -> None:
    provider = make_provider()

    with patch.object(provider, "_request_status", return_value=200), patch(
        "hermes.plugins.openbrain_memory.socket.create_connection",
        return_value=nullcontext(object()),
    ):
        assert provider.is_available() is True


def test_is_available_qdrant_down() -> None:
    provider = make_provider()

    with patch.object(provider, "_request_status", side_effect=OSError("down")), patch(
        "hermes.plugins.openbrain_memory.socket.create_connection",
        return_value=nullcontext(object()),
    ):
        assert provider.is_available() is False


def test_is_available_postgres_down() -> None:
    provider = make_provider()

    with patch.object(provider, "_request_status", return_value=200), patch(
        "hermes.plugins.openbrain_memory.socket.create_connection",
        side_effect=OSError("down"),
    ):
        assert provider.is_available() is False


def test_get_tool_schemas_returns_four_mcp_tools() -> None:
    provider = make_provider()

    schemas = provider.get_tool_schemas()

    assert [schema["name"] for schema in schemas] == [
        "brain_remember",
        "brain_recall",
        "brain_forget",
        "brain_list",
    ]

    for schema in schemas:
        assert isinstance(schema["description"], str)
        assert schema["inputSchema"]["type"] == "object"
        assert isinstance(schema["inputSchema"]["properties"], dict)
        if "required" in schema["inputSchema"]:
            assert isinstance(schema["inputSchema"]["required"], list)


@patch.object(OpenBrainMemoryProvider, "initialize", autospec=True)
def test_handle_tool_call_maps_tool_names_to_endpoints(mock_initialize) -> None:
    provider = make_provider()
    cases = [
        (
            "brain_remember",
            {"content": "remember this", "type": "fact"},
            "POST",
            "https://brain.example/v1/brain/remember",
            None,
        ),
        (
            "brain_recall",
            {"query": "find this", "limit": 3},
            "POST",
            "https://brain.example/v1/brain/recall",
            None,
        ),
        (
            "brain_forget",
            {"id": "123e4567-e89b-12d3-a456-426614174000"},
            "DELETE",
            "https://brain.example/v1/brain/forget/123e4567-e89b-12d3-a456-426614174000",
            None,
        ),
        (
            "brain_list",
            {"project": "corepy"},
            "GET",
            "https://brain.example/v1/brain/list",
            {"project": "corepy", "workspace_id": 73, "org": "lthn"},
        ),
    ]

    for name, args, method, url, params in cases:
        with patch.object(provider, "_request_json", return_value={"ok": True}) as mock_request:
            result = provider.handle_tool_call(name, args)

        assert result == {"ok": True}
        called_method, called_url = mock_request.call_args.args[:2]
        assert called_method == method
        assert called_url == url
        if params is None:
            assert mock_request.call_args.kwargs.get("params") is None
        else:
            assert mock_request.call_args.kwargs["params"] == params


@patch.object(OpenBrainMemoryProvider, "initialize", autospec=True)
def test_sync_turn_does_not_raise_on_non_200(mock_initialize) -> None:
    provider = make_provider()

    with patch.object(provider, "_dispatch_pending_write", return_value=False), patch.object(
        provider,
        "_request_json",
        return_value={"status": 500, "error": "service_unavailable"},
    ):
        provider.sync_turn({"content": "turn content", "project": "corepy"})
