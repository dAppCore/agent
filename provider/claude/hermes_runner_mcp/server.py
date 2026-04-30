# SPDX-License-Identifier: EUPL-1.2

from __future__ import annotations

import argparse
import json
import logging
import os
import selectors
import signal
import sys
import threading
from dataclasses import dataclass
from typing import Any, Literal
from urllib.parse import quote, urljoin

import httpx
from pydantic import BaseModel, Field, ValidationError

LOGGER = logging.getLogger("hermes_runner_mcp")
DEFAULT_HERMES_URL = "http://localhost:8642/"
RUNNER_NAME = "hermes-runner"
SUPPORTED_PROTOCOL_VERSIONS = (
    "2025-11-25",
    "2025-06-18",
    "2025-03-26",
    "2024-11-05",
)


class HermesAPIError(RuntimeError):
    """Raised when the Hermes gateway cannot satisfy a request."""


class DispatchRequest(BaseModel):
    """Validated input for the ``hermes_dispatch`` MCP tool.

    Holds the task prompt, the structured inputs map, and an optional
    subagent composition. ``gateway_payload()`` produces the JSON shape
    the Hermes gateway expects.

    Example::

        req = DispatchRequest(task="run tests", inputs={"repo": "go-store"})
        body = req.gateway_payload()
    """

    task: str
    inputs: dict[str, Any] = Field(default_factory=dict)
    agents: list[dict[str, Any]] | None = None

    def gateway_payload(self) -> dict[str, Any]:
        payload: dict[str, Any] = {
            "runner": RUNNER_NAME,
            "task": self.task,
            "inputs": self.inputs,
        }
        if self.agents is not None:
            payload["agents"] = self.agents
            payload["args"] = [
                "--agents",
                json.dumps(self.agents, separators=(",", ":"), sort_keys=True),
            ]
        return payload


class DispatchResult(BaseModel):
    """Successful result of a ``hermes_dispatch`` call: the run identifier
    plus a poll URL the caller can pass to ``hermes_status``.

    Example::

        DispatchResult(run_id="r-42", status_url="https://hermes/runs/r-42")
    """

    run_id: str
    status_url: str


class StatusResult(BaseModel):
    """Snapshot of a Hermes run's lifecycle state, exposed by the
    ``hermes_status`` MCP tool. ``state`` is one of queued / running /
    complete / failed; ``progress`` and ``last_event`` are runner-defined
    extensions for richer UIs.

    Example::

        StatusResult(state="running", progress={"step": 2, "total": 5})
    """

    state: Literal["queued", "running", "complete", "failed"]
    progress: Any = None
    last_event: Any = None


class FetchResult(BaseModel):
    """Final outputs of a completed Hermes run, exposed by the
    ``hermes_fetch`` MCP tool. ``output`` is the runner's primary result;
    ``artifacts`` is a list of side-channel files; ``log`` is any captured
    log tail.

    Example::

        FetchResult(output={"sha": "abc"}, artifacts=[], log="...")
    """

    output: Any = None
    artifacts: list[Any] = Field(default_factory=list)
    log: Any = None


@dataclass(frozen=True)
class ToolSpec:
    """Static MCP-tool descriptor — name, human description, JSON-schema
    input shape. Used to register tools onto the FastMCP server.

    Example::

        ToolSpec(name="hermes_dispatch", description="...", input_schema={...})
    """

    name: str
    description: str
    input_schema: dict[str, Any]


TOOL_SPECS = (
    ToolSpec(
        name="hermes_dispatch",
        description="Dispatch a Hermes runner job and return the run id plus status URL.",
        input_schema={
            "type": "object",
            "properties": {
                "task": {
                    "type": "string",
                    "description": "Task prompt or instruction for the Hermes run.",
                },
                "inputs": {
                    "type": "object",
                    "description": "Structured inputs passed through to Hermes.",
                    "default": {},
                },
                "agents": {
                    "type": "array",
                    "description": "Optional Hermes subagent composition passed as --agents JSON.",
                    "items": {
                        "type": "object",
                    },
                },
            },
            "required": ["task", "inputs"],
            "additionalProperties": False,
        },
    ),
    ToolSpec(
        name="hermes_status",
        description="Read the current queued/running/complete/failed state for a Hermes run.",
        input_schema={
            "type": "object",
            "properties": {
                "run_id": {
                    "type": "string",
                    "description": "Hermes run identifier returned by hermes_dispatch.",
                },
            },
            "required": ["run_id"],
            "additionalProperties": False,
        },
    ),
    ToolSpec(
        name="hermes_fetch",
        description="Fetch completed Hermes run output, artifact URLs, and a condensed log tail.",
        input_schema={
            "type": "object",
            "properties": {
                "run_id": {
                    "type": "string",
                    "description": "Hermes run identifier returned by hermes_dispatch.",
                },
            },
            "required": ["run_id"],
            "additionalProperties": False,
        },
    ),
)


def configure_logging() -> None:
    """Initialise the root logger to write INFO+ messages to stderr in the
    standard ``time level name: message`` format. Safe to call multiple times.

    Example::

        configure_logging()
    """

    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
        stream=sys.stderr,
    )


def install_signal_handlers(
    stop_event: threading.Event,
    *,
    exit_immediately: bool,
) -> None:
    """Wire SIGINT/SIGTERM to set ``stop_event`` so long-running loops can
    drain cleanly. When ``exit_immediately`` is true, the handler raises
    SystemExit on receipt instead of relying on the cooperative shutdown.

    Example::

        stop = threading.Event()
        install_signal_handlers(stop, exit_immediately=False)
    """

    def handle_signal(signum: int, _frame: Any) -> None:
        LOGGER.info("received signal %s, shutting down", signum)
        stop_event.set()
        if exit_immediately:
            raise SystemExit(0)

    signal.signal(signal.SIGINT, handle_signal)
    signal.signal(signal.SIGTERM, handle_signal)


class HermesGatewayClient:
    """HTTP client wrapper for the Hermes gateway. Owns a configured
    ``httpx.Client`` with the API key header pre-set and a base URL
    normalised to no trailing slash.

    Example::

        client = HermesGatewayClient("https://hermes.lthn.sh", api_key="...")
        try:
            run_id = client.dispatch(payload)
        finally:
            client.close()
    """

    def __init__(
        self,
        hermes_url: str,
        api_key: str | None = None,
        *,
        timeout: float = 30.0,
        transport: httpx.BaseTransport | None = None,
    ) -> None:
        self.base_url = self._normalise_base_url(hermes_url)
        self._client = httpx.Client(
            base_url=self.base_url,
            headers=self._build_headers(api_key),
            timeout=timeout,
            transport=transport,
        )

    def close(self) -> None:
        self._client.close()

    def dispatch(self, request: DispatchRequest) -> DispatchResult:
        payload = request.gateway_payload()
        response = self._request_json(
            "POST",
            ("dispatch", "runs"),
            json_body=payload,
        )
        run_id = self._require_string(
            response,
            ("run_id",),
            ("id",),
            ("data", "run_id"),
            ("data", "id"),
            ("result", "run_id"),
            ("result", "id"),
            ("run", "id"),
        )
        status_url = self._find_string(
            response,
            ("status_url",),
            ("data", "status_url"),
            ("result", "status_url"),
            ("run", "status_url"),
        )
        if status_url is None:
            status_url = urljoin(self.base_url, f"runs/{quote(run_id, safe='')}")
        return DispatchResult(run_id=run_id, status_url=status_url)

    def status(self, run_id: str) -> StatusResult:
        encoded_run_id = quote(run_id, safe="")
        response = self._request_json(
            "GET",
            (f"runs/{encoded_run_id}", f"status/{encoded_run_id}"),
        )
        raw_state = self._require_string(
            response,
            ("state",),
            ("status",),
            ("data", "state"),
            ("data", "status"),
            ("result", "state"),
            ("run", "state"),
        )
        state = self._normalise_state(raw_state)
        progress = self._find_value(
            response,
            ("progress",),
            ("data", "progress"),
            ("result", "progress"),
            ("run", "progress"),
        )
        last_event = self._find_value(
            response,
            ("last_event",),
            ("event",),
            ("data", "last_event"),
            ("data", "event"),
            ("result", "last_event"),
            ("run", "last_event"),
        )
        return StatusResult(state=state, progress=progress, last_event=last_event)

    def fetch(self, run_id: str) -> FetchResult:
        encoded_run_id = quote(run_id, safe="")
        response = self._request_json(
            "GET",
            (f"runs/{encoded_run_id}/fetch", f"fetch/{encoded_run_id}"),
        )
        output = self._find_value(
            response,
            ("output",),
            ("data", "output"),
            ("result", "output"),
        )
        artifacts = self._find_value(
            response,
            ("artifacts",),
            ("data", "artifacts"),
            ("result", "artifacts"),
        )
        log_tail = self._find_value(
            response,
            ("log",),
            ("tail",),
            ("data", "log"),
            ("data", "tail"),
            ("result", "log"),
        )
        if artifacts is None:
            normalised_artifacts: list[Any] = []
        elif isinstance(artifacts, list):
            normalised_artifacts = artifacts
        else:
            normalised_artifacts = [artifacts]
        return FetchResult(
            output=output,
            artifacts=normalised_artifacts,
            log=log_tail,
        )

    def _request_json(
        self,
        method: str,
        paths: tuple[str, ...],
        *,
        json_body: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        last_error: Exception | None = None

        # The exact gateway route shape is not specified, so we retry a small
        # set of conventional paths when the first one returns 404.
        for index, path in enumerate(paths):
            try:
                response = self._client.request(method, path, json=json_body)
                if response.status_code == 404 and index < len(paths) - 1:
                    continue
                response.raise_for_status()
                payload = response.json()
            except httpx.HTTPStatusError as exc:
                message = self._response_error_message(exc.response)
                raise HermesAPIError(
                    f"Hermes gateway {method} {exc.request.url} failed: {message}"
                ) from exc
            except httpx.RequestError as exc:
                raise HermesAPIError(
                    f"Hermes gateway request failed: {exc}"
                ) from exc
            except ValueError as exc:
                raise HermesAPIError(
                    f"Hermes gateway returned invalid JSON for {method} {path}"
                ) from exc

            if isinstance(payload, dict):
                return payload

            last_error = HermesAPIError(
                f"Hermes gateway returned non-object JSON for {method} {path}"
            )

        if last_error is None:
            last_error = HermesAPIError(
                f"Hermes gateway request failed for {method} {paths[0]}"
            )
        raise last_error

    def _response_error_message(self, response: httpx.Response) -> str:
        text = response.text.strip()
        if not text:
            return f"HTTP {response.status_code}"
        if len(text) > 300:
            return f"HTTP {response.status_code}: {text[:297]}..."
        return f"HTTP {response.status_code}: {text}"

    def _normalise_base_url(self, hermes_url: str) -> str:
        value = hermes_url.strip()
        if not value:
            raise ValueError("Hermes URL must not be empty")
        if not value.endswith("/"):
            value = f"{value}/"
        return value

    def _build_headers(self, api_key: str | None) -> dict[str, str]:
        headers = {
            "Accept": "application/json",
            "Content-Type": "application/json",
            "User-Agent": "hermes-runner-mcp/0.1.0",
        }
        if api_key:
            headers["Authorization"] = f"Bearer {api_key}"
            headers["X-API-Key"] = api_key
        return headers

    def _require_string(
        self,
        payload: dict[str, Any],
        *paths: tuple[str, ...],
    ) -> str:
        value = self._find_string(payload, *paths)
        if value is None:
            raise HermesAPIError(
                f"Missing required string field in Hermes gateway response: {paths}"
            )
        return value

    def _find_string(
        self,
        payload: dict[str, Any],
        *paths: tuple[str, ...],
    ) -> str | None:
        value = self._find_value(payload, *paths)
        if isinstance(value, str) and value:
            return value
        return None

    def _find_value(
        self,
        payload: dict[str, Any],
        *paths: tuple[str, ...],
    ) -> Any:
        for path in paths:
            current: Any = payload
            found = True
            for segment in path:
                if not isinstance(current, dict) or segment not in current:
                    found = False
                    break
                current = current[segment]
            if found:
                return current
        return None

    def _normalise_state(self, value: str) -> Literal["queued", "running", "complete", "failed"]:
        normalised = value.strip().lower()
        if normalised == "pending":
            normalised = "queued"
        if normalised == "completed":
            normalised = "complete"
        if normalised == "error":
            normalised = "failed"
        if normalised not in {"queued", "running", "complete", "failed"}:
            raise HermesAPIError(f"Unsupported Hermes run state: {value}")
        return normalised


class HermesToolHandler:
    """Glue between the MCP tool registration surface and a
    ``HermesGatewayClient``. Dispatches each tool name (hermes_dispatch /
    hermes_status / hermes_fetch) to the corresponding gateway call and
    formats the response into MCP-compatible result shapes.

    Example::

        client = HermesGatewayClient(...)
        handler = HermesToolHandler(client)
        tools = handler.list_tools()
    """

    def __init__(self, client: HermesGatewayClient) -> None:
        self.client = client

    def list_tools(self) -> list[dict[str, Any]]:
        return [
            {
                "name": spec.name,
                "description": spec.description,
                "inputSchema": spec.input_schema,
            }
            for spec in TOOL_SPECS
        ]

    def dispatch(
        self,
        task: str,
        inputs: dict[str, Any],
        agents: list[dict[str, Any]] | None = None,
    ) -> DispatchResult:
        request = DispatchRequest(task=task, inputs=inputs, agents=agents)
        return self.client.dispatch(request)

    def status(self, run_id: str) -> StatusResult:
        return self.client.status(run_id)

    def fetch(self, run_id: str) -> FetchResult:
        return self.client.fetch(run_id)

    def call(self, name: str, arguments: Any | None) -> dict[str, Any]:
        payload = {} if arguments is None else arguments
        if not isinstance(payload, dict):
            raise ValidationError.from_exception_data(
                "ToolArguments",
                [
                    {
                        "type": "dict_type",
                        "loc": ("arguments",),
                        "msg": "Tool arguments must be an object.",
                        "input": payload,
                    }
                ],
            )

        if name == "hermes_dispatch":
            request = DispatchRequest.model_validate(payload)
            return self.client.dispatch(request).model_dump(mode="json")
        if name == "hermes_status":
            run_id = _parse_run_identifier(payload)
            return self.client.status(run_id).model_dump(mode="json")
        if name == "hermes_fetch":
            run_id = _parse_run_identifier(payload)
            return self.client.fetch(run_id).model_dump(mode="json")
        raise KeyError(name)


class MinimalMCPServer:
    """Standalone JSON-RPC over stdio server speaking the MCP protocol.

    Used as a fallback when ``fastmcp`` isn't importable. Reads
    line-delimited JSON-RPC requests from stdin, dispatches them to the
    ``HermesToolHandler``, and writes responses back. Supports batch
    requests (JSON array) and exits cleanly when ``stop_event`` is set.

    Example::

        server = MinimalMCPServer(handler, stop_event=threading.Event())
        server.serve()
    """

    def __init__(self, handler: HermesToolHandler, stop_event: threading.Event) -> None:
        self.handler = handler
        self.stop_event = stop_event
        self._initialised = False

    def serve(self) -> None:
        selector = selectors.DefaultSelector()
        selector.register(sys.stdin, selectors.EVENT_READ)

        while not self.stop_event.is_set():
            events = selector.select(timeout=0.25)
            if not events:
                continue

            line = sys.stdin.readline()
            if line == "":
                break

            message = line.strip()
            if not message:
                continue

            try:
                payload = json.loads(message)
            except json.JSONDecodeError as exc:
                LOGGER.error("failed to decode JSON-RPC payload: %s", exc)
                continue

            if isinstance(payload, list):
                responses = [
                    response
                    for item in payload
                    for response in [self._handle_message(item)]
                    if response is not None
                ]
                if responses:
                    self._write_message(responses)
                continue

            response = self._handle_message(payload)
            if response is not None:
                self._write_message(response)

    def _handle_message(self, payload: Any) -> dict[str, Any] | None:
        if not isinstance(payload, dict):
            return self._error(None, -32600, "Invalid Request")

        method = payload.get("method")
        request_id = payload.get("id")

        if not isinstance(method, str):
            return None if "id" not in payload else self._error(request_id, -32600, "Invalid Request")

        params = payload.get("params")

        if method == "initialize":
            self._initialised = True
            requested = None
            if isinstance(params, dict):
                candidate = params.get("protocolVersion")
                if isinstance(candidate, str):
                    requested = candidate
            protocol_version = negotiate_protocol_version(requested)
            return self._result(
                request_id,
                {
                    "protocolVersion": protocol_version,
                    "capabilities": {
                        "tools": {
                            "listChanged": False,
                        }
                    },
                    "serverInfo": {
                        "name": "hermes-runner-mcp",
                        "version": "0.1.0",
                    },
                    "instructions": (
                        "Use hermes_dispatch to start remote Hermes work, "
                        "hermes_status to poll progress, and hermes_fetch to "
                        "retrieve final output and artifacts."
                    ),
                },
            )

        if method == "notifications/initialized":
            self._initialised = True
            return None

        if method == "ping":
            return self._result(request_id, {})

        if method == "exit":
            self.stop_event.set()
            return None

        if method == "notifications/cancelled":
            return None

        if not self._initialised:
            return self._error(
                request_id,
                -32002,
                "Server not initialised",
            )

        if method == "tools/list":
            return self._result(request_id, {"tools": self.handler.list_tools()})

        if method == "tools/call":
            if not isinstance(params, dict):
                return self._error(request_id, -32602, "Invalid params")
            name = params.get("name")
            arguments = params.get("arguments")
            if not isinstance(name, str):
                return self._error(request_id, -32602, "Missing tool name")
            try:
                tool_result = self.handler.call(name, arguments)
            except KeyError:
                return self._error(request_id, -32601, f"Unknown tool: {name}")
            except ValidationError as exc:
                return self._result(request_id, tool_error_result(format_validation_error(exc)))
            except HermesAPIError as exc:
                return self._result(request_id, tool_error_result(str(exc)))
            return self._result(request_id, tool_success_result(tool_result))

        return self._error(request_id, -32601, f"Method not found: {method}")

    def _result(self, request_id: Any, result: dict[str, Any]) -> dict[str, Any]:
        return {
            "jsonrpc": "2.0",
            "id": request_id,
            "result": result,
        }

    def _error(
        self,
        request_id: Any,
        code: int,
        message: str,
    ) -> dict[str, Any]:
        return {
            "jsonrpc": "2.0",
            "id": request_id,
            "error": {
                "code": code,
                "message": message,
            },
        }

    def _write_message(self, payload: dict[str, Any] | list[dict[str, Any]]) -> None:
        sys.stdout.write(json.dumps(payload, separators=(",", ":"), ensure_ascii=True))
        sys.stdout.write("\n")
        sys.stdout.flush()


def negotiate_protocol_version(requested: str | None) -> str:
    """Return the negotiated MCP protocol version: the client's requested
    version when supported, or the server's preferred fallback otherwise.

    Example::

        version = negotiate_protocol_version("2024-11-05")
    """

    if requested in SUPPORTED_PROTOCOL_VERSIONS:
        return requested
    return SUPPORTED_PROTOCOL_VERSIONS[0]


def _parse_run_identifier(payload: dict[str, Any]) -> str:
    """Extract a non-empty ``run_id`` string from a tool-call payload, or
    raise ``ValidationError`` describing the violation. Internal helper.
    """

    run_id = payload.get("run_id")
    if isinstance(run_id, str) and run_id:
        return run_id
    raise ValidationError.from_exception_data(
        "RunIdentifier",
        [
            {
                "type": "string_type",
                "loc": ("run_id",),
                "msg": "run_id must be a non-empty string.",
                "input": run_id,
            }
        ],
    )


def format_validation_error(exc: ValidationError) -> str:
    """Render a Pydantic ``ValidationError`` as a single human-readable
    string suitable for MCP error responses. Joins all field-level errors
    with semicolons in ``location: message`` form.

    Example::

        msg = format_validation_error(exc)
    """

    errors = []
    for error in exc.errors():
        location = ".".join(str(part) for part in error.get("loc", ()))
        message = error.get("msg", "Invalid value")
        if location:
            errors.append(f"{location}: {message}")
        else:
            errors.append(message)
    return "; ".join(errors) or "Invalid tool arguments"


def tool_success_result(payload: dict[str, Any]) -> dict[str, Any]:
    """Wrap a successful MCP tool result in the dual content/structured
    shape clients expect: a JSON-serialised text block plus the raw
    structured object.

    Example::

        result = tool_success_result({"run_id": "r-42"})
    """

    text = json.dumps(payload, separators=(",", ":"), ensure_ascii=True)
    return {
        "content": [
            {
                "type": "text",
                "text": text,
            }
        ],
        "structuredContent": payload,
        "isError": False,
    }


def tool_error_result(message: str) -> dict[str, Any]:
    """Wrap an MCP tool error in the standard text-content + isError shape.

    Example::

        err = tool_error_result("Hermes gateway timed out")
    """

    return {
        "content": [
            {
                "type": "text",
                "text": message,
            }
        ],
        "isError": True,
    }


def build_fastmcp_server(handler: HermesToolHandler) -> Any:
    """Construct a FastMCP server registered with the three Hermes tools
    (hermes_dispatch / hermes_status / hermes_fetch). Returns ``None`` when
    the optional ``mcp`` dependency is unavailable; callers should fall
    back to ``MinimalMCPServer`` in that case.

    Example::

        server = build_fastmcp_server(handler)
        if server is None:
            MinimalMCPServer(handler, stop_event).serve()
        else:
            server.run()
    """

    try:
        from typing import Annotated

        from mcp.server.fastmcp import FastMCP
        from mcp.types import CallToolResult, TextContent
    except ImportError:
        return None

    def make_result(payload: dict[str, Any], *, is_error: bool = False) -> Any:
        text = json.dumps(payload if not is_error else {"error": payload["error"]}, separators=(",", ":"), ensure_ascii=True)
        return CallToolResult(
            content=[TextContent(type="text", text=text)],
            structuredContent=None if is_error else payload,
            isError=is_error,
        )

    DispatchReturn = Annotated[CallToolResult, DispatchResult]
    StatusReturn = Annotated[CallToolResult, StatusResult]
    FetchReturn = Annotated[CallToolResult, FetchResult]

    try:
        server = FastMCP("Hermes Runner MCP", json_response=True)
    except TypeError:
        server = FastMCP("Hermes Runner MCP")

    @server.tool()
    def hermes_dispatch(
        task: str,
        inputs: dict[str, Any],
        agents: list[dict[str, Any]] | None = None,
    ) -> DispatchReturn:
        """Dispatch a Hermes run through the configured gateway."""
        try:
            payload = handler.dispatch(task=task, inputs=inputs, agents=agents).model_dump(mode="json")
        except (HermesAPIError, ValidationError) as exc:
            message = format_validation_error(exc) if isinstance(exc, ValidationError) else str(exc)
            return make_result({"error": message}, is_error=True)
        return make_result(payload)

    @server.tool()
    def hermes_status(run_id: str) -> StatusReturn:
        """Read the current state and progress of a Hermes run."""
        try:
            payload = handler.status(run_id).model_dump(mode="json")
        except HermesAPIError as exc:
            return make_result({"error": str(exc)}, is_error=True)
        return make_result(payload)

    @server.tool()
    def hermes_fetch(run_id: str) -> FetchReturn:
        """Fetch a completed Hermes run result, artifacts, and condensed log tail."""
        try:
            payload = handler.fetch(run_id).model_dump(mode="json")
        except HermesAPIError as exc:
            return make_result({"error": str(exc)}, is_error=True)
        return make_result(payload)

    return server


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    """Parse command-line arguments for the hermes-runner-mcp binary.

    Recognises ``--hermes-url`` and ``--api-key``; the latter falls back
    to the ``HERMES_API_KEY`` environment variable when omitted.

    Example::

        args = parse_args(["--hermes-url", "https://hermes.lthn.sh"])
    """

    parser = argparse.ArgumentParser(
        prog="hermes-runner-mcp",
        description="MCP stdio server for dispatching Hermes runner jobs.",
    )
    parser.add_argument(
        "--hermes-url",
        default=DEFAULT_HERMES_URL,
        help="Base URL for the Hermes gateway (default: %(default)s).",
    )
    parser.add_argument(
        "--api-key",
        default=os.environ.get("HERMES_API_KEY"),
        help="Hermes gateway API key. Defaults to HERMES_API_KEY.",
    )
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    """Entry point for the hermes-runner-mcp binary. Wires logging, the
    gateway client, the tool handler, and the chosen MCP server (FastMCP
    when available, MinimalMCPServer otherwise). Returns a process exit
    code (0 for clean shutdown, non-zero on fatal error).

    Example::

        sys.exit(main(sys.argv[1:]))
    """

    configure_logging()
    args = parse_args(argv)
    stop_event = threading.Event()
    client = HermesGatewayClient(args.hermes_url, args.api_key)
    handler = HermesToolHandler(client)
    exit_code = 0

    try:
        fastmcp_server = build_fastmcp_server(handler)
        if fastmcp_server is not None:
            install_signal_handlers(stop_event, exit_immediately=True)
            LOGGER.info("starting Hermes Runner MCP with official mcp SDK")
            fastmcp_server.run()
        else:
            install_signal_handlers(stop_event, exit_immediately=False)
            LOGGER.info("starting Hermes Runner MCP with minimal JSON-RPC stdio fallback")
            MinimalMCPServer(handler, stop_event).serve()
    except KeyboardInterrupt:
        LOGGER.info("shutting down after keyboard interrupt")
    finally:
        client.close()

    return exit_code


if __name__ == "__main__":
    raise SystemExit(main())
