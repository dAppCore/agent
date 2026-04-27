# SPDX-License-Identifier: EUPL-1.2

from __future__ import annotations

import importlib
import json
import shlex
import socket
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode, urlparse
from urllib.request import Request, urlopen

try:
    import requests  # type: ignore
except ImportError:  # pragma: no cover - exercised through fallbacks
    requests = None

try:
    import httpx  # type: ignore
except ImportError:  # pragma: no cover - exercised through fallbacks
    httpx = None


class OpenBrainTransportMixin:
    brain_url: str
    api_key: str
    qdrant_url: str
    pg_dsn: str

    def _qdrant_reachable(self) -> bool:
        try:
            status = self._request_status("GET", self._qdrant_probe_url(), timeout=1.5)
        except Exception:
            return False
        return 200 <= status < 500

    def _postgres_reachable(self) -> bool:
        host, port = self._postgres_target()
        if not host:
            return False

        timeout = 1.5

        try:
            if host.startswith("/"):
                socket_path = f"{host.rstrip('/')}/.s.PGSQL.{port}"
                with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as client:
                    client.settimeout(timeout)
                    client.connect(socket_path)
            else:
                with socket.create_connection((host, port), timeout=timeout):
                    pass
        except OSError:
            return False

        return True

    def _qdrant_probe_url(self) -> str:
        if not self.qdrant_url:
            return ""

        parsed = urlparse(self.qdrant_url)
        path = parsed.path or ""

        if not path or path == "/":
            return f"{self.qdrant_url}/collections"

        return self.qdrant_url

    def _postgres_target(self) -> tuple[str | None, int]:
        parsed = urlparse(self.pg_dsn)
        if parsed.scheme in {"postgres", "postgresql"}:
            host = parsed.hostname
            port = parsed.port or 5432
            return host, port

        parts: dict[str, str] = {}
        for token in shlex.split(self.pg_dsn):
            if "=" not in token:
                continue
            key, value = token.split("=", 1)
            parts[key.strip()] = value.strip()

        host = parts.get("host") or parts.get("hostaddr") or "localhost"
        port_value = parts.get("port", "5432")

        try:
            port = int(port_value)
        except ValueError:
            port = 5432

        if "," in host:
            host = host.split(",", 1)[0]

        return host, port

    def _load_core_spawn(self):
        try:
            task_module = importlib.import_module("core.task")
        except ImportError:
            return None
        return getattr(task_module, "spawn", None)

    def _clean_mapping(self, values: dict) -> dict:
        return {
            key: value
            for key, value in values.items()
            if value is not None and value != ""
        }

    def _brain_endpoint(self, suffix: str) -> str:
        return f"{self.brain_url}/v1/brain/{suffix.lstrip('/')}"

    def _auth_headers(self) -> dict[str, str]:
        return {
            "Accept": "application/json",
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json",
        }

    def _request_status(
        self,
        method: str,
        url: str,
        *,
        params: dict | None = None,
        json_body: dict | None = None,
        headers: dict | None = None,
        timeout: float = 5.0,
    ) -> int:
        status, _ = self._raw_request(
            method,
            url,
            params=params,
            json_body=json_body,
            headers=headers,
            timeout=timeout,
        )
        return status

    def _request_json(
        self,
        method: str,
        url: str,
        *,
        params: dict | None = None,
        json_body: dict | None = None,
        headers: dict | None = None,
        timeout: float = 5.0,
    ) -> dict:
        status, text = self._raw_request(
            method,
            url,
            params=params,
            json_body=json_body,
            headers=headers,
            timeout=timeout,
        )

        if not text:
            return {"status": status}

        try:
            payload = json.loads(text)
        except json.JSONDecodeError:
            return {"status": status, "data": text}

        if isinstance(payload, dict):
            payload.setdefault("status", status)
            return payload

        return {"status": status, "data": payload}

    def _raw_request(
        self,
        method: str,
        url: str,
        *,
        params: dict | None = None,
        json_body: dict | None = None,
        headers: dict | None = None,
        timeout: float = 5.0,
    ) -> tuple[int, str]:
        if requests is not None:
            response = requests.request(
                method,
                url,
                params=params,
                json=json_body,
                headers=headers,
                timeout=timeout,
            )
            return response.status_code, response.text

        if httpx is not None:
            response = httpx.request(
                method,
                url,
                params=params,
                json=json_body,
                headers=headers,
                timeout=timeout,
            )
            return response.status_code, response.text

        return self._urllib_request(
            method,
            url,
            params=params,
            json_body=json_body,
            headers=headers,
            timeout=timeout,
        )

    def _urllib_request(
        self,
        method: str,
        url: str,
        *,
        params: dict | None = None,
        json_body: dict | None = None,
        headers: dict | None = None,
        timeout: float = 5.0,
    ) -> tuple[int, str]:
        request_headers = dict(headers or {})
        request_url = url

        if params:
            query_string = urlencode(params, doseq=True)
            separator = "&" if "?" in request_url else "?"
            request_url = f"{request_url}{separator}{query_string}"

        data = None
        if json_body is not None:
            request_headers.setdefault("Content-Type", "application/json")
            data = json.dumps(json_body).encode("utf-8")

        request = Request(request_url, data=data, headers=request_headers, method=method)

        try:
            with urlopen(request, timeout=timeout) as response:
                return response.getcode(), response.read().decode("utf-8")
        except HTTPError as exc:
            return exc.code, exc.read().decode("utf-8")
        except URLError as exc:
            raise OSError(str(exc)) from exc
