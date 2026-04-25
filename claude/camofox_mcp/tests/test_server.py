# SPDX-License-Identifier: EUPL-1.2
# MockTransport fixtures in this module use literal http:// URLs for in-process tests only.
# Keep the inline NOSONAR markers paired with those mock-only endpoints.

from __future__ import annotations

import io
import json
import unittest

import httpx

from camofox_mcp.server import (
    CamofoxClient,
    CamofoxMcpApplication,
    MinimalStdioMcpServer,
    read_stdio_message,
)


def make_response(request: httpx.Request, payload: object, status_code: int = 200) -> httpx.Response:
    return httpx.Response(status_code, json=payload, request=request)


def make_bytes_response(request: httpx.Request, payload: bytes, status_code: int = 200) -> httpx.Response:
    return httpx.Response(status_code, content=payload, request=request)


class CamofoxMcpApplicationTests(unittest.TestCase):
    def make_app(self, handler) -> CamofoxMcpApplication:
        transport = httpx.MockTransport(handler)
        client = httpx.Client(transport=transport, base_url="http://camofox.local")  # NOSONAR python:S5332 - MockTransport base URL stays in-process.
        camofox = CamofoxClient(
            "http://camofox.local",  # NOSONAR python:S5332 - MockTransport client target stays in-process.
            "secret-token",
            user_id="agent1",
            session_key="session1",
            client=client,
        )
        self.addCleanup(camofox.close)
        return CamofoxMcpApplication(camofox)

    def test_navigate_returns_local_tab_handle_and_status(self) -> None:
        calls: list[tuple[str, str, dict[str, object], str | None]] = []

        def handler(request: httpx.Request) -> httpx.Response:
            body = json.loads(request.content.decode("utf-8")) if request.content else {}
            calls.append(
                (
                    request.method,
                    request.url.path,
                    body,
                    request.headers.get("Authorization"),
                )
            )
            if request.url.path == "/tabs":
                return make_response(request, {"tabId": "remote-abc", "status": "created"})
            if request.url.path == "/tabs/remote-abc/wait":
                return make_response(request, {"ok": True})
            raise AssertionError(f"unexpected request: {request.method} {request.url}")

        app = self.make_app(handler)

        result = app.dispatch_tool("navigate", {"url": "https://example.com"})

        self.assertEqual(result, {"tab_id": 1, "status": "ok"})
        self.assertEqual(
            calls,
            [
                (
                    "POST",
                    "/tabs",
                    {"userId": "agent1", "sessionKey": "session1", "url": "https://example.com"},
                    "Bearer secret-token",
                ),
                ("POST", "/tabs/remote-abc/wait", {"userId": "agent1"}, "Bearer secret-token"),
            ],
        )

    def test_read_page_uses_evaluate_endpoint(self) -> None:
        def handler(request: httpx.Request) -> httpx.Response:
            if request.url.path == "/tabs":
                return make_response(request, {"tabId": "remote-abc"})
            if request.url.path == "/tabs/remote-abc/wait":
                return make_response(request, {"ok": True})
            if request.url.path == "/tabs/remote-abc/evaluate":
                payload = json.loads(request.content.decode("utf-8"))
                self.assertEqual(payload["userId"], "agent1")
                self.assertIn("document.title", payload["expression"])
                return make_response(
                    request,
                    {
                        "ok": True,
                        "result": {
                            "text": "Hello world",
                            "url": "https://example.com",
                            "title": "Example Domain",
                        },
                    },
                )
            raise AssertionError(f"unexpected request: {request.method} {request.url}")

        app = self.make_app(handler)
        navigate = app.dispatch_tool("navigate", {"url": "https://example.com"})

        result = app.dispatch_tool("read_page", {"tab_id": navigate["tab_id"]})

        self.assertEqual(
            result,
            {
                "text": "Hello world",
                "url": "https://example.com",
                "title": "Example Domain",
            },
        )

    def test_screenshot_base64_encodes_png(self) -> None:
        image = b"\x89PNG\r\n\x1a\nmock"

        def handler(request: httpx.Request) -> httpx.Response:
            if request.url.path == "/tabs":
                return make_response(request, {"tabId": "remote-abc"})
            if request.url.path == "/tabs/remote-abc/wait":
                return make_response(request, {"ok": True})
            if request.url.path == "/tabs/remote-abc/screenshot":
                self.assertEqual(request.url.params["userId"], "agent1")
                self.assertEqual(request.url.params["fullPage"], "true")
                return make_bytes_response(request, image)
            raise AssertionError(f"unexpected request: {request.method} {request.url}")

        app = self.make_app(handler)
        navigate = app.dispatch_tool("navigate", {"url": "https://example.com"})

        result = app.dispatch_tool("screenshot", {"tab_id": navigate["tab_id"]})

        self.assertEqual(result, {"image_b64": "iVBORw0KGgptb2Nr"})

    def test_click_posts_selector(self) -> None:
        def handler(request: httpx.Request) -> httpx.Response:
            if request.url.path == "/tabs":
                return make_response(request, {"tabId": "remote-abc"})
            if request.url.path == "/tabs/remote-abc/wait":
                return make_response(request, {"ok": True})
            if request.url.path == "/tabs/remote-abc/click":
                self.assertEqual(
                    json.loads(request.content.decode("utf-8")),
                    {"userId": "agent1", "selector": "#submit"},
                )
                return make_response(request, {"ok": True})
            raise AssertionError(f"unexpected request: {request.method} {request.url}")

        app = self.make_app(handler)
        navigate = app.dispatch_tool("navigate", {"url": "https://example.com"})

        result = app.dispatch_tool("click", {"tab_id": navigate["tab_id"], "selector": "#submit"})

        self.assertEqual(result, {"ok": True})

    def test_fill_posts_type_request(self) -> None:
        def handler(request: httpx.Request) -> httpx.Response:
            if request.url.path == "/tabs":
                return make_response(request, {"tabId": "remote-abc"})
            if request.url.path == "/tabs/remote-abc/wait":
                return make_response(request, {"ok": True})
            if request.url.path == "/tabs/remote-abc/type":
                self.assertEqual(
                    json.loads(request.content.decode("utf-8")),
                    {"userId": "agent1", "selector": "#username", "text": "snider"},
                )
                return make_response(request, {"ok": True})
            raise AssertionError(f"unexpected request: {request.method} {request.url}")

        app = self.make_app(handler)
        navigate = app.dispatch_tool("navigate", {"url": "https://example.com"})

        result = app.dispatch_tool(
            "fill",
            {"tab_id": navigate["tab_id"], "selector": "#username", "value": "snider"},
        )

        self.assertEqual(result, {"ok": True})

    def test_close_tab_posts_delete_and_unregisters_handle(self) -> None:
        def handler(request: httpx.Request) -> httpx.Response:
            if request.url.path == "/tabs":
                return make_response(request, {"tabId": "remote-abc"})
            if request.url.path == "/tabs/remote-abc/wait":
                return make_response(request, {"ok": True})
            if request.method == "DELETE" and request.url.path == "/tabs/remote-abc":
                self.assertEqual(json.loads(request.content.decode("utf-8")), {"userId": "agent1"})
                return make_response(request, {"ok": True})
            raise AssertionError(f"unexpected request: {request.method} {request.url}")

        app = self.make_app(handler)
        navigate = app.dispatch_tool("navigate", {"url": "https://example.com"})

        result = app.dispatch_tool("close_tab", {"tab_id": navigate["tab_id"]})

        self.assertEqual(result, {"ok": True})
        with self.assertRaisesRegex(Exception, "Unknown tab_id"):
            app.dispatch_tool("read_page", {"tab_id": navigate["tab_id"]})


class MinimalStdioMcpServerTests(unittest.TestCase):
    def test_stdio_server_dispatches_initialize_tools_list_and_call(self) -> None:
        def handler(request: httpx.Request) -> httpx.Response:
            if request.url.path == "/tabs":
                return make_response(request, {"tabId": "remote-abc", "status": "created"})
            if request.url.path == "/tabs/remote-abc/wait":
                return make_response(request, {"ok": True})
            raise AssertionError(f"unexpected request: {request.method} {request.url}")

        transport = httpx.MockTransport(handler)
        client = httpx.Client(transport=transport, base_url="http://camofox.local")  # NOSONAR python:S5332 - MockTransport base URL stays in-process.
        camofox = CamofoxClient(
            "http://camofox.local",  # NOSONAR python:S5332 - MockTransport client target stays in-process.
            user_id="agent1",
            session_key="session1",
            client=client,
        )
        app = CamofoxMcpApplication(camofox)
        self.addCleanup(app.close)
        server = MinimalStdioMcpServer(app)

        stdin = io.BytesIO(
            b'{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"claude","version":"1.0"}}}\n'
            b'{"jsonrpc":"2.0","method":"notifications/initialized"}\n'
            b'{"jsonrpc":"2.0","id":2,"method":"tools/list"}\n'
            b'{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"navigate","arguments":{"url":"https://example.com"}}}\n'
        )
        stdout = io.BytesIO()

        server.serve(stdin, stdout)

        responses = [json.loads(line) for line in stdout.getvalue().decode("utf-8").splitlines() if line.strip()]
        self.assertEqual(responses[0]["result"]["protocolVersion"], "2025-03-26")
        self.assertEqual(responses[1]["result"]["tools"][0]["name"], "navigate")
        self.assertEqual(responses[2]["result"]["content"][0]["type"], "text")
        self.assertIn('"tab_id":1', responses[2]["result"]["content"][0]["text"])

    def test_content_length_framing_is_accepted(self) -> None:
        payload = json.dumps({"jsonrpc": "2.0", "id": 1, "method": "ping"}).encode("utf-8")
        framed = io.BytesIO(f"Content-Length: {len(payload)}\r\n\r\n".encode("ascii") + payload)

        message, framing = read_stdio_message(framed)

        self.assertEqual(framing, "content-length")
        self.assertEqual(json.loads(message), {"jsonrpc": "2.0", "id": 1, "method": "ping"})


if __name__ == "__main__":
    unittest.main()
