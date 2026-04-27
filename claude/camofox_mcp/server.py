# SPDX-License-Identifier: EUPL-1.2

from __future__ import annotations

import argparse
import base64
import json
import logging
import os
import signal
import sys
import threading
from collections.abc import Mapping
from typing import Any, BinaryIO

import httpx
from pydantic import BaseModel, ConfigDict, Field, ValidationError

from . import __version__

try:
    import asyncio
    import mcp.server.stdio as mcp_stdio
    import mcp.types as mcp_types
    from mcp.server.lowlevel import NotificationOptions, Server as McpServer
    from mcp.server.models import InitializationOptions

    MCP_SDK_AVAILABLE = True
except ImportError:  # pragma: no cover - exercised implicitly in local sandbox.
    asyncio = None
    mcp_stdio = None
    mcp_types = None
    NotificationOptions = None
    McpServer = None
    InitializationOptions = None
    MCP_SDK_AVAILABLE = False


LOGGER = logging.getLogger("camofox_mcp")
SERVER_NAME = "camofox-mcp"
DEFAULT_CAMOFOX_URL = "http://localhost:8099/"
SUPPORTED_PROTOCOL_VERSIONS = ("2025-11-25", "2025-06-18", "2025-03-26")
READ_PAGE_EXPRESSION = (
    "(() => {"
    "const root = document.body ?? document.documentElement;"
    "return {"
    "title: document.title ?? '',"
    "url: window.location.href ?? '',"
    "text: root ? (root.innerText ?? root.textContent ?? '') : ''"
    "};"
    "})()"
)


class JsonRpcError(Exception):
    """JSON-RPC protocol error."""

    def __init__(self, code: int, message: str, data: Any | None = None) -> None:
        super().__init__(message)
        self.code = code
        self.message = message
        self.data = data


class ToolExecutionError(Exception):
    """Tool execution failure that should be surfaced as an MCP tool error."""


class NavigateArgs(BaseModel):
    model_config = ConfigDict(extra="forbid", str_strip_whitespace=True)

    url: str = Field(min_length=1)


class TabArgs(BaseModel):
    model_config = ConfigDict(extra="forbid")

    tab_id: int = Field(ge=1)


class ClickArgs(TabArgs):
    model_config = ConfigDict(extra="forbid", str_strip_whitespace=True)

    selector: str = Field(min_length=1)


class FillArgs(ClickArgs):
    value: str


class ToolDefinition:
    """Simple tool registry entry."""

    def __init__(self, name: str, description: str, model: type[BaseModel], handler_name: str) -> None:
        self.name = name
        self.description = description
        self.model = model
        self.handler_name = handler_name

    def schema(self) -> dict[str, Any]:
        return {
            "name": self.name,
            "description": self.description,
            "inputSchema": self.model.model_json_schema(),
        }


class CamofoxClient:
    """Thin HTTP wrapper around the Camofox browser server."""

    def __init__(
        self,
        base_url: str,
        api_key: str | None = None,
        *,
        user_id: str | None = None,
        session_key: str | None = None,
        client: httpx.Client | None = None,
    ) -> None:
        pid = os.getpid()
        self.user_id = user_id or os.getenv("CAMOFOX_USER_ID") or f"claude-code-{pid}"
        self.session_key = session_key or f"claude-code-{pid}"
        self.api_key = api_key
        self.base_url = base_url.rstrip("/") or base_url
        headers = {"Accept": "application/json"}
        if api_key:
            headers["Authorization"] = f"Bearer {api_key}"
        self._client = client or httpx.Client(base_url=self.base_url, headers=headers, timeout=30.0)

    def request_json(
        self,
        method: str,
        path: str,
        *,
        json_body: dict[str, Any] | None = None,
        params: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        try:
            response = self._client.request(
                method,
                path,
                json=json_body,
                params=params,
                headers=self._request_headers(),
            )
            response.raise_for_status()
        except httpx.HTTPStatusError as exc:
            detail = exc.response.text.strip() or str(exc)
            raise ToolExecutionError(f"Camofox API returned HTTP {exc.response.status_code}: {detail}") from exc
        except httpx.HTTPError as exc:
            raise ToolExecutionError(f"Camofox API request failed: {exc}") from exc

        if not response.content:
            return {}

        try:
            return response.json()
        except ValueError as exc:
            raise ToolExecutionError(f"Camofox API returned invalid JSON for {method} {path}") from exc

    def request_bytes(self, path: str, *, params: dict[str, Any] | None = None) -> bytes:
        try:
            response = self._client.request("GET", path, params=params, headers=self._request_headers())
            response.raise_for_status()
        except httpx.HTTPStatusError as exc:
            detail = exc.response.text.strip() or str(exc)
            raise ToolExecutionError(f"Camofox API returned HTTP {exc.response.status_code}: {detail}") from exc
        except httpx.HTTPError as exc:
            raise ToolExecutionError(f"Camofox API request failed: {exc}") from exc

        return response.content

    def close(self) -> None:
        self._client.close()

    def _request_headers(self) -> dict[str, str]:
        headers = {"Accept": "application/json"}
        if self.api_key:
            headers["Authorization"] = f"Bearer {self.api_key}"
        return headers


class CamofoxMcpApplication:
    """Business logic for the six exposed MCP tools."""

    TOOLS = (
        ToolDefinition("navigate", "Open a URL in a new Camofox tab.", NavigateArgs, "_navigate"),
        ToolDefinition("read_page", "Read the current page text, URL, and title for a tab.", TabArgs, "_read_page"),
        ToolDefinition("screenshot", "Capture a PNG screenshot for a tab and return it as base64.", TabArgs, "_screenshot"),
        ToolDefinition("click", "Click a CSS selector inside a tab.", ClickArgs, "_click"),
        ToolDefinition("fill", "Fill a CSS selector inside a tab with text.", FillArgs, "_fill"),
        ToolDefinition("close_tab", "Close a tab.", TabArgs, "_close_tab"),
    )

    def __init__(self, client: CamofoxClient) -> None:
        self.client = client
        self._lock = threading.Lock()
        self._next_tab_id = 1
        self._remote_tabs: dict[int, str] = {}
        self._tool_map = {tool.name: tool for tool in self.TOOLS}

    def close(self) -> None:
        self.client.close()

    def tool_schemas(self) -> list[dict[str, Any]]:
        return [tool.schema() for tool in self.TOOLS]

    def dispatch_tool(self, name: str, arguments: Mapping[str, Any] | None) -> dict[str, Any]:
        tool = self._tool_map.get(name)
        if tool is None:
            raise JsonRpcError(-32601, f"Unknown tool: {name}")

        try:
            validated = tool.model.model_validate(arguments or {})
        except ValidationError as exc:
            raise JsonRpcError(-32602, exc.json(include_url=False)) from exc

        handler = getattr(self, tool.handler_name)
        return handler(validated)

    def _navigate(self, args: NavigateArgs) -> dict[str, Any]:
        response = self.client.request_json(
            "POST",
            "/tabs",
            json_body={
                "userId": self.client.user_id,
                "sessionKey": self.client.session_key,
                "url": args.url,
            },
        )
        remote_tab_id = self._extract_remote_tab_id(response)
        local_tab_id = self._register_tab(remote_tab_id)
        status = str(self._extract_first(response, ("status", "state", "message")) or "ok")

        try:
            wait_response = self.client.request_json(
                "POST",
                f"/tabs/{remote_tab_id}/wait",
                json_body={"userId": self.client.user_id},
            )
        except ToolExecutionError as exc:
            LOGGER.debug("navigate wait failed for tab %s: %s", remote_tab_id, exc)
        else:
            status = str(self._extract_first(wait_response, ("status", "state", "message")) or "ok")

        return {"tab_id": local_tab_id, "status": status}

    def _read_page(self, args: TabArgs) -> dict[str, Any]:
        remote_tab_id = self._resolve_tab(args.tab_id)

        try:
            response = self.client.request_json(
                "POST",
                f"/tabs/{remote_tab_id}/evaluate",
                json_body={
                    "userId": self.client.user_id,
                    "expression": READ_PAGE_EXPRESSION,
                },
            )
            result = response.get("result", response)
            if not isinstance(result, Mapping):
                raise ToolExecutionError("Camofox evaluate response did not contain page metadata")
            return {
                "text": str(result.get("text", "")),
                "url": str(result.get("url", "")),
                "title": str(result.get("title", "")),
            }
        except ToolExecutionError as exc:
            LOGGER.debug("read_page evaluate failed for tab %s: %s", remote_tab_id, exc)

        snapshot = self.client.request_json(
            "GET",
            f"/tabs/{remote_tab_id}/snapshot",
            params={"userId": self.client.user_id},
        )
        return {
            "text": str(self._extract_first(snapshot, ("snapshot", "text")) or ""),
            "url": str(self._extract_first(snapshot, ("url",)) or ""),
            "title": str(self._extract_first(snapshot, ("title", "pageTitle")) or ""),
        }

    def _screenshot(self, args: TabArgs) -> dict[str, Any]:
        remote_tab_id = self._resolve_tab(args.tab_id)
        image = self.client.request_bytes(
            f"/tabs/{remote_tab_id}/screenshot",
            params={"userId": self.client.user_id, "fullPage": "true"},
        )
        return {"image_b64": base64.b64encode(image).decode("ascii")}

    def _click(self, args: ClickArgs) -> dict[str, Any]:
        remote_tab_id = self._resolve_tab(args.tab_id)
        response = self.client.request_json(
            "POST",
            f"/tabs/{remote_tab_id}/click",
            json_body={"userId": self.client.user_id, "selector": args.selector},
        )
        return {"ok": self._extract_ok(response)}

    def _fill(self, args: FillArgs) -> dict[str, Any]:
        remote_tab_id = self._resolve_tab(args.tab_id)
        response = self.client.request_json(
            "POST",
            f"/tabs/{remote_tab_id}/type",
            json_body={
                "userId": self.client.user_id,
                "selector": args.selector,
                "text": args.value,
            },
        )
        return {"ok": self._extract_ok(response)}

    def _close_tab(self, args: TabArgs) -> dict[str, Any]:
        remote_tab_id = self._resolve_tab(args.tab_id)
        response = self.client.request_json(
            "DELETE",
            f"/tabs/{remote_tab_id}",
            json_body={"userId": self.client.user_id},
        )
        with self._lock:
            self._remote_tabs.pop(args.tab_id, None)
        return {"ok": self._extract_ok(response)}

    def _register_tab(self, remote_tab_id: str) -> int:
        with self._lock:
            local_tab_id = self._next_tab_id
            self._next_tab_id += 1
            self._remote_tabs[local_tab_id] = remote_tab_id
        return local_tab_id

    def _resolve_tab(self, local_tab_id: int) -> str:
        with self._lock:
            remote_tab_id = self._remote_tabs.get(local_tab_id)
        if remote_tab_id is None:
            raise ToolExecutionError(f"Unknown tab_id: {local_tab_id}")
        return remote_tab_id

    @staticmethod
    def _extract_ok(response: Mapping[str, Any]) -> bool:
        value = response.get("ok")
        if isinstance(value, bool):
            return value
        success = response.get("success")
        if isinstance(success, bool):
            return success
        return True

    @classmethod
    def _extract_remote_tab_id(cls, payload: Any) -> str:
        value = cls._extract_first(payload, ("tabId", "tab_id", "targetId", "target_id", "id"))
        if value in (None, ""):
            raise ToolExecutionError("Camofox did not return a tab identifier")
        return str(value)

    @classmethod
    def _extract_first(cls, payload: Any, keys: tuple[str, ...]) -> Any | None:
        if isinstance(payload, Mapping):
            for key in keys:
                if key in payload and payload[key] not in (None, ""):
                    return payload[key]
            for value in payload.values():
                found = cls._extract_first(value, keys)
                if found not in (None, ""):
                    return found
            return None
        if isinstance(payload, list):
            for item in payload:
                found = cls._extract_first(item, keys)
                if found not in (None, ""):
                    return found
        return None


class MinimalStdioMcpServer:
    """Small MCP stdio server used when the official SDK is unavailable."""

    def __init__(self, app: CamofoxMcpApplication) -> None:
        self.app = app
        self.protocol_version = SUPPORTED_PROTOCOL_VERSIONS[0]
        self.initialised = False
        self._output_framing = "line"

    def serve(self, reader: BinaryIO, writer: BinaryIO) -> None:
        while True:
            payload, framing = read_stdio_message(reader)
            if payload is None:
                break
            self._output_framing = framing or self._output_framing

            try:
                message = json.loads(payload)
            except json.JSONDecodeError as exc:
                response = jsonrpc_error(None, -32700, f"Parse error: {exc.msg}")
                write_stdio_message(writer, response, framing=self._output_framing)
                continue

            response = self.handle_message(message)
            if response is None:
                continue
            write_stdio_message(writer, response, framing=self._output_framing)

    def handle_message(self, message: Any) -> dict[str, Any] | list[dict[str, Any]] | None:
        if isinstance(message, list):
            if not message:
                return jsonrpc_error(None, -32600, "Invalid Request")

            responses: list[dict[str, Any]] = []
            for item in message:
                response = self._handle_single(item)
                if response is not None:
                    responses.append(response)
            return responses or None

        return self._handle_single(message)

    def _handle_single(self, message: Any) -> dict[str, Any] | None:
        if not isinstance(message, Mapping):
            return jsonrpc_error(None, -32600, "Invalid Request")

        request_id = message.get("id")
        method = message.get("method")
        if not isinstance(method, str):
            return jsonrpc_error(request_id, -32600, "Invalid Request")

        try:
            result = self._dispatch(method, message.get("params"))
        except JsonRpcError as exc:
            return jsonrpc_error(request_id, exc.code, exc.message, exc.data)
        except ToolExecutionError as exc:
            if request_id is None:
                return None
            return jsonrpc_success(request_id, self._tool_error_result(str(exc)))
        except Exception as exc:  # pragma: no cover - defensive guard.
            LOGGER.exception("Unexpected MCP server failure")
            return jsonrpc_error(request_id, -32603, f"Internal error: {exc}")

        if request_id is None:
            return None
        return jsonrpc_success(request_id, result)

    def _dispatch(self, method: str, params: Any) -> dict[str, Any]:
        if method == "initialize":
            if not isinstance(params, Mapping):
                raise JsonRpcError(-32602, "initialize requires params")
            requested = params.get("protocolVersion")
            self.protocol_version = negotiate_protocol_version(requested)
            return {
                "protocolVersion": self.protocol_version,
                "capabilities": {"tools": {"listChanged": False}},
                "serverInfo": {"name": SERVER_NAME, "version": __version__},
                "instructions": "Expose the Camofox browser HTTP API as MCP tools for Claude Code.",
            }

        if method == "notifications/initialized":
            self.initialised = True
            return {}

        if method == "ping":
            return {}

        if not self.initialised:
            raise JsonRpcError(-32002, "Server has not completed initialization")

        if method == "tools/list":
            return {"tools": self.app.tool_schemas()}

        if method == "tools/call":
            if not isinstance(params, Mapping):
                raise JsonRpcError(-32602, "tools/call requires params")
            name = params.get("name")
            arguments = params.get("arguments")
            if not isinstance(name, str):
                raise JsonRpcError(-32602, "tools/call requires a tool name")
            result = self.app.dispatch_tool(name, arguments if isinstance(arguments, Mapping) else {})
            return self._tool_success_result(result)

        if method == "resources/list":
            return {"resources": []}

        if method == "resources/templates/list":
            return {"resourceTemplates": []}

        if method == "prompts/list":
            return {"prompts": []}

        if method == "logging/setLevel":
            return {}

        raise JsonRpcError(-32601, f"Method not found: {method}")

    def _tool_success_result(self, result: dict[str, Any]) -> dict[str, Any]:
        payload = {
            "content": [{"type": "text", "text": json.dumps(result, ensure_ascii=False, separators=(",", ":"))}],
            "isError": False,
        }
        if self.protocol_version >= "2025-06-18":
            payload["structuredContent"] = result
        return payload

    @staticmethod
    def _tool_error_result(message: str) -> dict[str, Any]:
        return {"content": [{"type": "text", "text": message}], "isError": True}


def build_argument_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Camofox MCP stdio server")
    parser.add_argument("--camofox-url", default=DEFAULT_CAMOFOX_URL, help="Base URL for the Camofox HTTP API")
    parser.add_argument(
        "--api-key",
        default=os.getenv("CAMOFOX_API_KEY"),
        help="Bearer token for the Camofox API (defaults to CAMOFOX_API_KEY)",
    )
    return parser


def configure_logging() -> None:
    logging.basicConfig(
        level=os.getenv("CAMOFOX_LOG_LEVEL", "INFO").upper(),
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
        stream=sys.stderr,
    )


def install_signal_handlers() -> None:
    def _handle_signal(signum: int, _frame: Any) -> None:
        LOGGER.info("Received signal %s, shutting down", signum)
        raise KeyboardInterrupt

    signal.signal(signal.SIGINT, _handle_signal)
    signal.signal(signal.SIGTERM, _handle_signal)


def negotiate_protocol_version(requested: Any) -> str:
    if isinstance(requested, str) and requested in SUPPORTED_PROTOCOL_VERSIONS:
        return requested
    return SUPPORTED_PROTOCOL_VERSIONS[0]


def jsonrpc_success(request_id: Any, result: dict[str, Any]) -> dict[str, Any]:
    return {"jsonrpc": "2.0", "id": request_id, "result": result}


def jsonrpc_error(request_id: Any, code: int, message: str, data: Any | None = None) -> dict[str, Any]:
    error = {"code": code, "message": message}
    if data is not None:
        error["data"] = data
    return {"jsonrpc": "2.0", "id": request_id, "error": error}


def read_stdio_message(reader: BinaryIO) -> tuple[str | None, str | None]:
    while True:
        line = reader.readline()
        if line == b"":
            return None, None
        if line in (b"\n", b"\r\n"):
            continue

        if line.lower().startswith(b"content-length:"):
            try:
                content_length = int(line.split(b":", 1)[1].strip())
            except (IndexError, ValueError) as exc:
                raise JsonRpcError(-32700, f"Invalid Content-Length header: {line!r}") from exc

            while True:
                header = reader.readline()
                if header == b"":
                    return None, "content-length"
                if header in (b"\n", b"\r\n"):
                    break

            payload = reader.read(content_length)
            if len(payload) != content_length:
                return None, "content-length"
            return payload.decode("utf-8"), "content-length"

        return line.strip().decode("utf-8"), "line"


def write_stdio_message(writer: BinaryIO, message: Any, *, framing: str = "line") -> None:
    payload = json.dumps(message, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
    if framing == "content-length":
        writer.write(f"Content-Length: {len(payload)}\r\n\r\n".encode("ascii"))
        writer.write(payload)
    else:
        writer.write(payload + b"\n")

    flush = getattr(writer, "flush", None)
    if callable(flush):
        flush()


async def run_sdk_server(app: CamofoxMcpApplication) -> None:
    if not MCP_SDK_AVAILABLE or asyncio is None or mcp_stdio is None or mcp_types is None:
        raise RuntimeError("MCP SDK is not available")

    server = McpServer(SERVER_NAME)

    @server.list_tools()
    async def _list_tools() -> list[Any]:
        return [
            mcp_types.Tool(
                name=tool["name"],
                description=tool["description"],
                inputSchema=tool["inputSchema"],
            )
            for tool in app.tool_schemas()
        ]

    @server.call_tool()
    async def _call_tool(name: str, arguments: dict[str, Any] | None) -> Any:
        try:
            result = app.dispatch_tool(name, arguments or {})
            return mcp_types.CallToolResult(
                content=[
                    mcp_types.TextContent(
                        type="text",
                        text=json.dumps(result, ensure_ascii=False, separators=(",", ":")),
                    )
                ],
                structuredContent=result,
                isError=False,
            )
        except (JsonRpcError, ToolExecutionError) as exc:
            return mcp_types.CallToolResult(
                content=[mcp_types.TextContent(type="text", text=str(exc))],
                isError=True,
            )

    async with mcp_stdio.stdio_server() as (read_stream, write_stream):
        await server.run(
            read_stream,
            write_stream,
            InitializationOptions(
                server_name=SERVER_NAME,
                server_version=__version__,
                capabilities=server.get_capabilities(
                    notification_options=NotificationOptions(),
                    experimental_capabilities={},
                ),
            ),
        )


def run_fallback_server(app: CamofoxMcpApplication) -> None:
    server = MinimalStdioMcpServer(app)
    server.serve(sys.stdin.buffer, sys.stdout.buffer)


def main(argv: list[str] | None = None) -> int:
    configure_logging()
    install_signal_handlers()
    args = build_argument_parser().parse_args(argv)

    app = CamofoxMcpApplication(CamofoxClient(args.camofox_url, args.api_key))
    try:
        LOGGER.info(
            "Starting %s against %s using %s transport",
            SERVER_NAME,
            args.camofox_url,
            "official MCP SDK" if MCP_SDK_AVAILABLE else "fallback JSON-RPC",
        )
        if MCP_SDK_AVAILABLE:
            asyncio.run(run_sdk_server(app))
        else:
            run_fallback_server(app)
    except KeyboardInterrupt:
        LOGGER.info("Shutdown requested")
    finally:
        app.close()

    return 0


if __name__ == "__main__":  # pragma: no cover
    raise SystemExit(main())
