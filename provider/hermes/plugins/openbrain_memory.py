# SPDX-License-Identifier: EUPL-1.2

from __future__ import annotations

import atexit
import threading
from typing import Any

from .openbrain_common import OpenBrainTransportMixin


VALID_MEMORY_TYPES = [
    "fact",
    "decision",
    "observation",
    "convention",
    "research",
    "plan",
    "bug",
    "architecture",
    "documentation",
    "service",
    "pattern",
    "context",
    "procedure",
]


class OpenBrainMemoryProvider(OpenBrainTransportMixin):
    """Hermes MemoryProvider that writes new memories into OpenBrain.

    Extends ``OpenBrainTransportMixin`` for reachability + transport. The
    provider validates the memory ``type`` against the allow-listed set
    declared in this module before persisting to the brain backend.

    Example::

        provider = OpenBrainMemoryProvider(
            brain_url="https://brain.lthn.sh",
            api_key=os.environ["OPENBRAIN_KEY"],
            qdrant_url="http://qdrant.lthn.sh:6333",
            pg_dsn="postgres://...",
            workspace_id=42,
        )
        provider.remember("decision", "...content...", tags=["rfc"])
    """

    def __init__(
        self,
        brain_url: str,
        api_key: str,
        qdrant_url: str,
        pg_dsn: str,
        workspace_id: int,
        org: str | None = None,
    ) -> None:
        self.brain_url = brain_url.rstrip("/")
        self.api_key = api_key
        self.qdrant_url = qdrant_url.rstrip("/")
        self.pg_dsn = pg_dsn
        self.workspace_id = workspace_id
        self.org = org
        self._initialised = False
        self._spawn = None
        self._pending_writes: list[Any] = []
        self._pending_lock = threading.Lock()

    def is_available(self) -> bool:
        return self._qdrant_reachable() and self._postgres_reachable()

    def initialize(self) -> None:
        if self._initialised:
            return

        self._spawn = self._load_core_spawn()
        atexit.register(self.on_session_end)
        self._initialised = True

    def get_tool_schemas(self) -> list[dict]:
        return [
            {
                "name": "brain_remember",
                "description": (
                    "Store a memory in the shared OpenBrain knowledge store. "
                    "Use this for durable decisions, observations, conventions, "
                    "research, plans, bugs, or architecture notes."
                ),
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "content": {
                            "type": "string",
                            "description": "The knowledge to remember.",
                            "maxLength": 50000,
                        },
                        "type": {
                            "type": "string",
                            "description": "Memory type classification.",
                            "enum": VALID_MEMORY_TYPES,
                        },
                        "tags": {
                            "type": "array",
                            "items": {"type": "string"},
                            "description": "Optional tags for categorisation.",
                        },
                        "project": {
                            "type": "string",
                            "description": "Optional project scope.",
                        },
                        "org": {
                            "type": "string",
                            "description": "Optional organisation scope.",
                        },
                        "confidence": {
                            "type": "number",
                            "description": "Confidence score from 0.0 to 1.0.",
                            "minimum": 0.0,
                            "maximum": 1.0,
                        },
                        "supersedes": {
                            "type": "string",
                            "format": "uuid",
                            "description": "UUID of an older memory this entry replaces.",
                        },
                        "expires_in": {
                            "type": "integer",
                            "description": "Hours until the memory expires.",
                            "minimum": 1,
                        },
                    },
                    "required": ["content", "type"],
                },
            },
            {
                "name": "brain_recall",
                "description": (
                    "Semantic search across the shared OpenBrain knowledge store. "
                    "Returns memories ranked by similarity to the query."
                ),
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "query": {
                            "type": "string",
                            "description": "Natural-language search query.",
                            "maxLength": 2000,
                        },
                        "limit": {
                            "type": "integer",
                            "description": "Maximum results to return.",
                            "minimum": 1,
                            "maximum": 20,
                            "default": 5,
                        },
                        "top_k": {
                            "type": "integer",
                            "description": "Alias for limit used by the Brain API.",
                            "minimum": 1,
                            "maximum": 20,
                        },
                        "workspace_id": {
                            "type": "integer",
                            "description": "Workspace scope for the recall request.",
                            "minimum": 1,
                        },
                        "org": {
                            "type": "string",
                            "description": "Optional organisation filter.",
                        },
                        "project": {
                            "type": "string",
                            "description": "Optional project filter.",
                        },
                        "type": {
                            "description": "Optional memory type filter.",
                            "oneOf": [
                                {"type": "string", "enum": VALID_MEMORY_TYPES},
                                {
                                    "type": "array",
                                    "items": {
                                        "type": "string",
                                        "enum": VALID_MEMORY_TYPES,
                                    },
                                },
                            ],
                        },
                        "keywords": {
                            "type": "array",
                            "items": {"type": "string"},
                            "description": "Keywords that should be present in matching memories.",
                        },
                        "boost_keywords": {
                            "type": "array",
                            "items": {"type": "string"},
                            "description": "Keywords that should receive additional ranking weight.",
                        },
                        "agent_id": {
                            "type": "string",
                            "description": "Optional originating-agent filter.",
                        },
                        "min_confidence": {
                            "type": "number",
                            "description": "Minimum confidence threshold.",
                            "minimum": 0.0,
                            "maximum": 1.0,
                        },
                    },
                    "required": ["query"],
                },
            },
            {
                "name": "brain_forget",
                "description": "Remove a memory from the shared OpenBrain knowledge store by UUID.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "id": {
                            "type": "string",
                            "format": "uuid",
                            "description": "UUID of the memory to remove.",
                        },
                        "reason": {
                            "type": "string",
                            "description": "Optional reason for forgetting this memory.",
                            "maxLength": 500,
                        },
                    },
                    "required": ["id"],
                },
            },
            {
                "name": "brain_list",
                "description": (
                    "List memories in the shared OpenBrain knowledge store. "
                    "Supports filtering by project, type, agent, and limit."
                ),
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "workspace_id": {
                            "type": "integer",
                            "description": "Workspace scope for the list request.",
                            "minimum": 1,
                        },
                        "org": {
                            "type": "string",
                            "description": "Optional organisation scope.",
                        },
                        "project": {
                            "type": "string",
                            "description": "Optional project scope.",
                        },
                        "type": {
                            "type": "string",
                            "description": "Optional memory type filter.",
                            "enum": VALID_MEMORY_TYPES,
                        },
                        "agent_id": {
                            "type": "string",
                            "description": "Optional originating-agent filter.",
                        },
                        "limit": {
                            "type": "integer",
                            "description": "Maximum results to return.",
                            "minimum": 1,
                            "maximum": 100,
                            "default": 20,
                        },
                    },
                },
            },
        ]

    def handle_tool_call(self, name: str, args: dict) -> dict:
        self.initialize()
        request_args = dict(args or {})

        if name == "brain_remember":
            payload = self._with_context_defaults(request_args, include_workspace=False)
            return self._request_json(
                "POST",
                self._brain_endpoint("remember"),
                json_body=payload,
                headers=self._auth_headers(),
            )

        if name == "brain_recall":
            payload = self._with_context_defaults(request_args)
            if "limit" in payload and "top_k" not in payload:
                payload["top_k"] = payload["limit"]
            return self._request_json(
                "POST",
                self._brain_endpoint("recall"),
                json_body=payload,
                headers=self._auth_headers(),
            )

        if name == "brain_forget":
            memory_id = request_args.get("id")
            if not memory_id:
                raise ValueError("brain_forget requires an id")
            payload = self._clean_mapping({"reason": request_args.get("reason")})
            return self._request_json(
                "DELETE",
                self._brain_endpoint(f"forget/{memory_id}"),
                json_body=payload or None,
                headers=self._auth_headers(),
            )

        if name == "brain_list":
            params = self._with_context_defaults(request_args)
            return self._request_json(
                "GET",
                self._brain_endpoint("list"),
                params=params,
                headers=self._auth_headers(),
            )

        raise ValueError(f"Unsupported tool call: {name}")

    def sync_turn(self, turn: dict) -> None:
        self.initialize()
        payload = self._build_turn_memory(turn)

        if self._dispatch_pending_write(payload):
            return

        try:
            self._request_json(
                "POST",
                self._brain_endpoint("remember"),
                json_body=payload,
                headers=self._auth_headers(),
                timeout=1.0,
            )
        except Exception:
            return

    def system_prompt_block(self) -> str:
        return (
            "Librarian keeps compact shared memory for the workspace: durable facts, "
            "decisions, observations, conventions, plans, bugs, architecture notes, "
            "and session context stored with project scope, tags, and confidence."
        )

    def on_session_end(self) -> None:
        with self._pending_lock:
            pending = list(self._pending_writes)
            self._pending_writes.clear()

        for handle in pending:
            self._await_pending(handle)

    def _dispatch_pending_write(self, payload: dict) -> bool:
        if not callable(self._spawn):
            return False

        handle = None

        try:
            handle = self._spawn(self._post_turn_memory, payload)
        except TypeError:
            try:
                handle = self._spawn(lambda: self._post_turn_memory(payload))
            except Exception:
                return False
        except Exception:
            return False

        if handle is not None:
            with self._pending_lock:
                self._pending_writes.append(handle)

        return True

    def _await_pending(self, handle: Any) -> None:
        waiters = (
            ("result", (2.0,)),
            ("join", (2.0,)),
            ("wait", (2.0,)),
        )

        for name, args in waiters:
            waiter = getattr(handle, name, None)
            if callable(waiter):
                try:
                    waiter(*args)
                except TypeError:
                    waiter()
                except Exception:
                    pass
                return

    def _post_turn_memory(self, payload: dict) -> None:
        self._request_json(
            "POST",
            self._brain_endpoint("remember"),
            json_body=payload,
            headers=self._auth_headers(),
            timeout=1.0,
        )

    def _build_turn_memory(self, turn: dict) -> dict:
        tags = list(turn.get("tags") or [])
        if "hermes" not in tags:
            tags.append("hermes")
        if "session-turn" not in tags:
            tags.append("session-turn")

        payload = {
            "content": json.dumps(turn, sort_keys=True, default=str),
            "type": turn.get("type") or "context",
            "tags": tags,
            "project": turn.get("project"),
            "org": turn.get("org", self.org),
            "confidence": turn.get("confidence", 0.6),
            "workspace_id": turn.get("workspace_id", self.workspace_id),
        }

        return self._clean_mapping(payload)

    def _with_context_defaults(self, values: dict, include_workspace: bool = True) -> dict:
        payload = dict(values)
        if include_workspace and "workspace_id" not in payload:
            payload["workspace_id"] = self.workspace_id
        if self.org and "org" not in payload:
            payload["org"] = self.org
        return self._clean_mapping(payload)

