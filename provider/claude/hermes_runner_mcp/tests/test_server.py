# SPDX-License-Identifier: EUPL-1.2
# MockTransport fixtures in this module use literal http:// URLs for reserved test hosts only.
# Keep the inline NOSONAR markers paired with those mock-only endpoints.

from __future__ import annotations

import json
import sys
import unittest
from pathlib import Path

import httpx

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from hermes_runner_mcp.server import DispatchRequest, HermesGatewayClient


class HermesGatewayClientTests(unittest.TestCase):
    def setUp(self) -> None:
        self.responses: dict[tuple[str, str], httpx.Response] = {}
        self.requests: list[httpx.Request] = []
        self.transport = httpx.MockTransport(self._handle_request)
        self.client = HermesGatewayClient(
            "http://hermes.example/",  # NOSONAR python:S5332 - MockTransport fixture target uses a reserved test host.
            "secret-key",
            transport=self.transport,
        )

    def tearDown(self) -> None:
        self.client.close()

    def _handle_request(self, request: httpx.Request) -> httpx.Response:
        self.requests.append(request)
        response = self.responses.get((request.method, request.url.path))
        if response is None:
            return httpx.Response(404, json={"error": "not_found"})
        return response

    def test_dispatch_posts_runner_payload_and_agents_args(self) -> None:
        self.responses[("POST", "/dispatch")] = httpx.Response(
            200,
            json={
                "run_id": "run-123",
                "status_url": "http://hermes.example/runs/run-123",  # NOSONAR python:S5332 - MockTransport response fixture uses a reserved test host.
            },
        )

        result = self.client.dispatch(
            DispatchRequest(
                task="Investigate ticket 13-6",
                inputs={"ticket": "13-6", "repo": "corepy"},
                agents=[{"name": "planner", "mode": "strict"}],
            )
        )

        self.assertEqual(result.run_id, "run-123")
        self.assertEqual(result.status_url, "http://hermes.example/runs/run-123")  # NOSONAR python:S5332 - MockTransport assertion uses a reserved test host.
        self.assertEqual(len(self.requests), 1)
        request = self.requests[0]
        self.assertEqual(request.headers["authorization"], "Bearer secret-key")
        self.assertEqual(request.headers["x-api-key"], "secret-key")

        payload = json.loads(request.content.decode("utf-8"))
        self.assertEqual(payload["runner"], "hermes-runner")
        self.assertEqual(payload["task"], "Investigate ticket 13-6")
        self.assertEqual(payload["inputs"], {"ticket": "13-6", "repo": "corepy"})
        self.assertEqual(payload["agents"], [{"name": "planner", "mode": "strict"}])
        self.assertEqual(
            payload["args"],
            [
                "--agents",
                json.dumps(
                    [{"name": "planner", "mode": "strict"}],
                    separators=(",", ":"),
                    sort_keys=True,
                ),
            ],
        )

    def test_status_reads_nested_payload_and_normalises_completed(self) -> None:
        self.responses[("GET", "/runs/run-456")] = httpx.Response(
            200,
            json={
                "data": {
                    "state": "completed",
                    "progress": 100,
                    "last_event": "finished cleanly",
                }
            },
        )

        result = self.client.status("run-456")

        self.assertEqual(result.state, "complete")
        self.assertEqual(result.progress, 100)
        self.assertEqual(result.last_event, "finished cleanly")

    def test_fetch_returns_output_artifacts_and_log(self) -> None:
        self.responses[("GET", "/runs/run-789/fetch")] = httpx.Response(
            200,
            json={
                "output": {"summary": "done"},
                "artifacts": ["https://artifacts.example/run-789/report.json"],
                "log": "last lines",
            },
        )

        result = self.client.fetch("run-789")

        self.assertEqual(result.output, {"summary": "done"})
        self.assertEqual(
            result.artifacts,
            ["https://artifacts.example/run-789/report.json"],
        )
        self.assertEqual(result.log, "last lines")


if __name__ == "__main__":
    unittest.main()
