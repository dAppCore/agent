# SPDX-License-Identifier: EUPL-1.2

from __future__ import annotations

import math
import sys
from collections.abc import Iterable

try:
    import networkx as nx  # type: ignore
except ImportError:  # pragma: no cover - exercised through manual fallback
    nx = None

from .openbrain_common import OpenBrainTransportMixin


class OpenBrainContextEngine(OpenBrainTransportMixin):
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
        self._similarity_threshold = 0.6

    def is_available(self) -> bool:
        return self._qdrant_reachable() and self._postgres_reachable()

    def initialize(self) -> None:
        if self._initialised:
            return

        self._spawn = self._load_core_spawn()
        self._initialised = True

    def compress(
        self,
        turns: list[dict],
        *,
        token_budget: int,
        query_hint: str | None = None,
        top_k: int = 20,
        candidate_pool: int = 200,
    ) -> list[dict]:
        ordered_turns = list(turns)
        if len(ordered_turns) <= 2:
            return ordered_turns

        self.initialize()

        budget = max(int(token_budget), 0)
        total_tokens = self._estimate_total_tokens(ordered_turns)
        if total_tokens <= budget:
            return ordered_turns

        query = self._derive_query(ordered_turns, query_hint)
        if not query:
            return self._naive_head_tail(ordered_turns, budget)

        try:
            candidates = self._recall_candidates(query, max(int(candidate_pool), 1))
        except Exception as exc:
            self._warn(f"OpenBrain recall failed in compress(); falling back to head+tail truncation: {exc}")
            return self._naive_head_tail(ordered_turns, budget)

        if not candidates:
            self._warn("OpenBrain recall returned no candidates in compress(); falling back to head+tail truncation.")
            return self._naive_head_tail(ordered_turns, budget)

        anchor_indices = {0, len(ordered_turns) - 1}
        nodes = self._build_turn_nodes(ordered_turns) + self._build_candidate_nodes(candidates)
        centrality, affinity = self._graph_scores(nodes)
        ranked_turns = self._rank_turn_indices(ordered_turns, centrality, affinity, anchor_indices)

        rank_selected = self._select_ranked_turns(ranked_turns, anchor_indices, max(int(top_k), 0))
        rank_selected = self._trim_selection_to_budget(
            rank_selected,
            ranked_turns,
            ordered_turns,
            budget,
            anchor_indices,
        )
        budget_selected = self._select_budget_turns(ranked_turns, ordered_turns, budget, anchor_indices)

        if len(budget_selected) > len(rank_selected):
            selected = budget_selected
        else:
            selected = rank_selected

        return [turn for index, turn in enumerate(ordered_turns) if index in selected]

    def system_prompt_block(self) -> str:
        return (
            "Librarian/Cartographer compresses chat history with centrality-ranked OpenBrain recall, "
            "keeping the system anchor and current user turn while dropping low-centrality turns "
            "when the token budget is tight."
        )

    def _recall_candidates(self, query: str, candidate_pool: int) -> list[dict]:
        payload = {
            "query": query,
            "top_k": candidate_pool,
            "filter": self._clean_mapping(
                {
                    "workspace_id": self.workspace_id,
                    "org": self.org,
                }
            ),
        }
        response = self._request_json(
            "POST",
            self._brain_endpoint("recall"),
            json_body=payload,
            headers=self._auth_headers(),
        )
        status = int(response.get("status", 200))
        if status >= 400:
            raise OSError(f"recall returned status {status}")
        return self._extract_candidates(response)

    def _extract_candidates(self, response: dict) -> list[dict]:
        sources: list[object] = [response]

        data = response.get("data")
        if isinstance(data, dict):
            sources.append(data)

        items: list[dict] = []
        for source in sources:
            if not isinstance(source, dict):
                continue

            for key in ("memories", "results", "items", "matches", "data"):
                value = source.get(key)
                if isinstance(value, list):
                    items.extend(item for item in value if isinstance(item, dict))

        return [self._normalise_candidate(item, index) for index, item in enumerate(items)]

    def _normalise_candidate(self, item: dict, index: int) -> dict:
        payload = item.get("payload")
        if not isinstance(payload, dict):
            payload = {}

        candidate_id = item.get("id") or item.get("memory_id") or item.get("uuid") or f"memory:{index}"
        text = (
            payload.get("content")
            or item.get("content")
            or payload.get("text")
            or item.get("text")
            or ""
        )

        return {
            "id": str(candidate_id),
            "text": self._stringify_text(text),
            "payload": payload,
            "score": self._float_value(item.get("score"), item.get("similarity"), default=0.0),
            "vector": self._coerce_vector(
                item.get("vector")
                or item.get("embedding")
                or payload.get("vector")
                or payload.get("embedding")
            ),
        }

    def _build_turn_nodes(self, turns: list[dict]) -> list[dict]:
        nodes: list[dict] = []
        for index, turn in enumerate(turns):
            node_id = turn.get("id") or turn.get("uuid") or f"turn:{index}"
            nodes.append(
                {
                    "node_id": str(node_id),
                    "turn_index": index,
                    "kind": "turn",
                    "text": self._turn_text(turn),
                    "vector": self._coerce_vector(turn.get("vector") or turn.get("embedding")),
                    "score": 0.0,
                }
            )
        return nodes

    def _build_candidate_nodes(self, candidates: list[dict]) -> list[dict]:
        return [
            {
                "node_id": candidate["id"],
                "turn_index": None,
                "kind": "memory",
                "text": candidate["text"],
                "vector": candidate["vector"],
                "score": candidate["score"],
            }
            for candidate in candidates
        ]

    def _graph_scores(self, nodes: list[dict]) -> tuple[dict[str, float], dict[str, float]]:
        adjacency = {node["node_id"]: set() for node in nodes}
        affinity = {node["node_id"]: 0.0 for node in nodes}

        for left_index in range(len(nodes)):
            left = nodes[left_index]
            for right_index in range(left_index + 1, len(nodes)):
                right = nodes[right_index]
                similarity = self._node_similarity(left, right)
                if similarity < self._similarity_threshold:
                    continue

                adjacency[left["node_id"]].add(right["node_id"])
                adjacency[right["node_id"]].add(left["node_id"])

                if left["kind"] != right["kind"]:
                    affinity[left["node_id"]] = max(affinity[left["node_id"]], similarity)
                    affinity[right["node_id"]] = max(affinity[right["node_id"]], similarity)

        centrality = self._degree_centrality(nodes, adjacency)
        return centrality, affinity

    def _degree_centrality(self, nodes: list[dict], adjacency: dict[str, set[str]]) -> dict[str, float]:
        if nx is not None:
            graph = nx.Graph()
            for node in nodes:
                graph.add_node(node["node_id"])
            for node_id, edges in adjacency.items():
                for edge in edges:
                    graph.add_edge(node_id, edge)
            return dict(nx.degree_centrality(graph))

        normaliser = max(len(nodes) - 1, 1)
        return {node_id: len(edges) / normaliser for node_id, edges in adjacency.items()}

    def _rank_turn_indices(
        self,
        turns: list[dict],
        centrality: dict[str, float],
        affinity: dict[str, float],
        anchor_indices: set[int],
    ) -> list[int]:
        ranked: list[tuple[float, float, float, int]] = []

        for index, turn in enumerate(turns):
            if index in anchor_indices:
                continue

            node_id = str(turn.get("id") or turn.get("uuid") or f"turn:{index}")
            ranked.append(
                (
                    -centrality.get(node_id, 0.0),
                    -affinity.get(node_id, 0.0),
                    self._estimate_tokens(turn),
                    index,
                )
            )

        ranked.sort()
        return [index for _, _, _, index in ranked]

    def _select_ranked_turns(self, ranked_turns: list[int], anchor_indices: set[int], top_k: int) -> set[int]:
        selected = set(anchor_indices)
        for index in ranked_turns[:top_k]:
            selected.add(index)
        return selected

    def _select_budget_turns(
        self,
        ranked_turns: list[int],
        turns: list[dict],
        token_budget: int,
        anchor_indices: set[int],
    ) -> set[int]:
        selected = set(anchor_indices)
        used_tokens = self._selection_tokens(selected, turns)

        for index in ranked_turns:
            turn_tokens = self._estimate_tokens(turns[index])
            if used_tokens + turn_tokens > token_budget and selected:
                continue
            selected.add(index)
            used_tokens += turn_tokens

        return selected

    def _trim_selection_to_budget(
        self,
        selected: set[int],
        ranked_turns: list[int],
        turns: list[dict],
        token_budget: int,
        anchor_indices: set[int],
    ) -> set[int]:
        trimmed = set(selected)
        if self._selection_tokens(trimmed, turns) <= token_budget:
            return trimmed

        removable = [index for index in reversed(ranked_turns) if index in trimmed and index not in anchor_indices]
        for index in removable:
            trimmed.remove(index)
            if self._selection_tokens(trimmed, turns) <= token_budget:
                break

        return trimmed

    def _selection_tokens(self, selected: set[int], turns: list[dict]) -> int:
        return sum(self._estimate_tokens(turns[index]) for index in selected)

    def _naive_head_tail(self, turns: list[dict], token_budget: int) -> list[dict]:
        if not turns:
            return []

        if len(turns) <= 2:
            return list(turns)

        selected = {0, len(turns) - 1}
        used_tokens = self._selection_tokens(selected, turns)

        for index in range(len(turns) - 2, 0, -1):
            turn_tokens = self._estimate_tokens(turns[index])
            if used_tokens + turn_tokens > token_budget and selected:
                break
            selected.add(index)
            used_tokens += turn_tokens

        return [turn for index, turn in enumerate(turns) if index in selected]

    def _derive_query(self, turns: list[dict], query_hint: str | None) -> str:
        if query_hint and query_hint.strip():
            return query_hint.strip()

        last_text = self._turn_text(turns[-1])
        if last_text:
            return last_text

        if len(turns) >= 2:
            previous = self._turn_text(turns[-2])
            parts = [part for part in (previous, last_text) if part]
            return "\n".join(parts)

        return ""

    def _estimate_total_tokens(self, turns: list[dict]) -> int:
        return sum(self._estimate_tokens(turn) for turn in turns)

    def _estimate_tokens(self, turn: dict) -> int:
        return len(self._turn_text(turn)) // 4

    def _turn_text(self, turn: dict) -> str:
        content = turn.get("content")
        if isinstance(content, str):
            return content
        if isinstance(content, list):
            parts = [self._stringify_text(item) for item in content]
            return " ".join(part for part in parts if part)
        if content is None:
            return ""
        return self._stringify_text(content)

    def _stringify_text(self, value: object) -> str:
        if isinstance(value, str):
            return value
        if value is None:
            return ""
        try:
            return json.dumps(value, sort_keys=True, default=str)
        except TypeError:
            return str(value)

    def _coerce_vector(self, value: object) -> list[float] | None:
        if not isinstance(value, Iterable) or isinstance(value, (str, bytes, dict)):
            return None

        vector: list[float] = []
        for item in value:
            try:
                vector.append(float(item))
            except (TypeError, ValueError):
                return None

        return vector or None

    def _node_similarity(self, left: dict, right: dict) -> float:
        left_vector = left.get("vector")
        right_vector = right.get("vector")
        if left_vector and right_vector:
            cosine = self._cosine_similarity(left_vector, right_vector)
            if cosine is not None:
                return cosine

        return self._jaccard_similarity(left.get("text", ""), right.get("text", ""))

    def _cosine_similarity(self, left: list[float], right: list[float]) -> float | None:
        if len(left) != len(right) or not left:
            return None

        numerator = sum(a * b for a, b in zip(left, right))
        left_norm = math.sqrt(sum(a * a for a in left))
        right_norm = math.sqrt(sum(b * b for b in right))
        if math.isclose(left_norm, 0.0, abs_tol=1e-12) or math.isclose(right_norm, 0.0, abs_tol=1e-12):
            return None

        return numerator / (left_norm * right_norm)

    def _jaccard_similarity(self, left_text: str, right_text: str) -> float:
        left_tokens = self._token_set(left_text)
        right_tokens = self._token_set(right_text)
        if not left_tokens or not right_tokens:
            return 0.0

        union = left_tokens | right_tokens
        if not union:
            return 0.0

        return len(left_tokens & right_tokens) / len(union)

    def _token_set(self, text: str) -> set[str]:
        lowered = text.lower()
        cleaned = "".join(character if character.isalnum() else " " for character in lowered)
        return {token for token in cleaned.split() if token}

    def _float_value(self, *values: object, default: float) -> float:
        for value in values:
            try:
                return float(value)
            except (TypeError, ValueError):
                continue
        return default

    def _warn(self, message: str) -> None:
        print(message, file=sys.stderr)
