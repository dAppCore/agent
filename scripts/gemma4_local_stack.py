#!/usr/bin/env python3
# SPDX-License-Identifier: EUPL-1.2

from __future__ import annotations

import argparse
import json
import os
import signal
import subprocess
import sys
import time
import urllib.error
import urllib.request
from dataclasses import dataclass, replace
from pathlib import Path
from typing import Iterable


DEFAULT_SERVER = "/private/tmp/core-agent-mlx-vlm/bin/mlx_vlm.server"
DEFAULT_APC_PATH = "/private/tmp/mlx-vlm-apc"
DEFAULT_LOG_DIR = "/private/tmp/core-agent-gemma4-stack"


@dataclass(frozen=True)
class Lane:
    name: str
    role: str
    model: str
    port: int
    max_kv_size: int
    max_tokens: int
    apc_blocks: int
    apc_disk_gb: int


LANES = {
    "main26": Lane(
        name="main26",
        role="main",
        model="mlx-community/gemma-4-26b-a4b-it-4bit",
        port=8001,
        max_kv_size=262144,
        max_tokens=2048,
        apc_blocks=20000,
        apc_disk_gb=32,
    ),
    "helper-e4b": Lane(
        name="helper-e4b",
        role="helper",
        model="mlx-community/gemma-4-e4b-it-mxfp8",
        port=8005,
        max_kv_size=131072,
        max_tokens=1024,
        apc_blocks=10000,
        apc_disk_gb=8,
    ),
    "helper-e2b": Lane(
        name="helper-e2b",
        role="helper",
        model="mlx-community/gemma-4-e2b-it-4bit",
        port=8004,
        max_kv_size=131072,
        max_tokens=1024,
        apc_blocks=10000,
        apc_disk_gb=8,
    ),
}


def lane_env(lane: Lane, apc_path: str) -> dict[str, str]:
    env = os.environ.copy()
    env.update(
        {
            "APC_ENABLED": "1",
            "APC_NUM_BLOCKS": str(lane.apc_blocks),
            "APC_BLOCK_SIZE": "16",
            "APC_LAYER_MAJOR_MEMORY_MIN_TOKENS": "50000",
            "APC_DISK_PATH": apc_path,
            "APC_DISK_MAX_GB": str(lane.apc_disk_gb),
            "APC_DISK_SHARD_MAX_BLOCKS": "256",
        }
    )
    return env


def lane_command(server: str, lane: Lane, host: str) -> list[str]:
    return [
        server,
        "--host",
        host,
        "--port",
        str(lane.port),
        "--model",
        lane.model,
        "--max-kv-size",
        str(lane.max_kv_size),
        "--max-tokens",
        str(lane.max_tokens),
    ]


def health_url(host: str, lane: Lane) -> str:
    return f"http://{host}:{lane.port}/health"


def cache_stats_url(host: str, lane: Lane) -> str:
    return f"http://{host}:{lane.port}/v1/cache/stats"


def read_json(url: str, timeout: float = 2.0) -> dict | None:
    request = urllib.request.Request(url)
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            body = response.read().decode("utf-8")
    except (urllib.error.URLError, TimeoutError):
        return None
    try:
        return json.loads(body)
    except json.JSONDecodeError:
        return {"raw": body}


def wait_ready(host: str, lane: Lane, timeout: float) -> bool:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        try:
            with urllib.request.urlopen(health_url(host, lane), timeout=2.0):
                return True
        except (urllib.error.URLError, TimeoutError):
            time.sleep(1.0)
    return False


def selected_lanes(args: argparse.Namespace) -> list[Lane]:
    main_base = LANES["main26"]
    helper_base = LANES[args.helper]
    main = replace(
        main_base,
        port=args.main_port if args.main_port is not None else main_base.port,
        max_kv_size=args.main_context,
        max_tokens=args.main_max_tokens,
    )
    helper = replace(
        helper_base,
        port=args.helper_port if args.helper_port is not None else helper_base.port,
        max_kv_size=args.helper_context,
        max_tokens=args.helper_max_tokens,
    )
    lanes = []
    if not args.helper_only:
        lanes.append(main)
    if not args.main_only:
        lanes.append(helper)
    return lanes


def print_commands(args: argparse.Namespace, lanes: Iterable[Lane]) -> None:
    for lane in lanes:
        env = lane_env(lane, args.apc_path)
        env_prefix = " ".join(
            f"{key}={env[key]}"
            for key in (
                "APC_ENABLED",
                "APC_NUM_BLOCKS",
                "APC_BLOCK_SIZE",
                "APC_LAYER_MAJOR_MEMORY_MIN_TOKENS",
                "APC_DISK_PATH",
                "APC_DISK_MAX_GB",
                "APC_DISK_SHARD_MAX_BLOCKS",
            )
        )
        command = " ".join(lane_command(args.server, lane, args.host))
        print(f"{lane.name}: {env_prefix} {command}")


def print_opencode(args: argparse.Namespace) -> None:
    main = replace(
        LANES["main26"],
        port=args.main_port if args.main_port is not None else LANES["main26"].port,
        max_kv_size=args.main_context,
        max_tokens=args.main_max_tokens,
    )
    helper_base = LANES[args.helper]
    helper = replace(
        helper_base,
        port=args.helper_port if args.helper_port is not None else helper_base.port,
        max_kv_size=args.helper_context,
        max_tokens=args.helper_max_tokens,
    )
    print("# CoreAgent/OpenCode profile overrides for this stack")
    print(f"export CORE_OPENCODE_GEMMA4_MLX_AGENTIC_BASE_URL=http://{args.host}:{main.port}/v1")
    print(f"export CORE_OPENCODE_GEMMA4_MLX_AGENTIC_MODEL={main.model}")
    if args.helper == "helper-e4b":
        print(f"export CORE_OPENCODE_GEMMA4_MLX_E4B_BASE_URL=http://{args.host}:{helper.port}/v1")
        print(f"export CORE_OPENCODE_GEMMA4_MLX_E4B_MODEL={helper.model}")
    else:
        print(f"export CORE_OPENCODE_GEMMA4_MLX_E2B_BASE_URL=http://{args.host}:{helper.port}/v1")
        print(f"export CORE_OPENCODE_GEMMA4_MLX_E2B_MODEL={helper.model}")
    print()
    print("# Main synthesis lane:")
    print('core agentic dispatch --agent opencode:gemma4-mlx-agentic --repo core/agent --task "..."')
    print("# Helper/sub-agent lane:")
    profile = "opencode:gemma4-mlx-e4b" if args.helper == "helper-e4b" else "opencode:gemma4-mlx-e2b"
    print(f'core agentic dispatch --agent {profile} --repo core/agent --task "..."')


def serve(args: argparse.Namespace) -> int:
    lanes = selected_lanes(args)
    if args.dry_run:
        print_commands(args, lanes)
        return 0

    log_dir = Path(args.log_dir)
    log_dir.mkdir(parents=True, exist_ok=True)
    processes: list[tuple[Lane, subprocess.Popen]] = []

    def terminate(_signum: int, _frame) -> None:
        for _, process in processes:
            if process.poll() is None:
                process.terminate()

    signal.signal(signal.SIGINT, terminate)
    signal.signal(signal.SIGTERM, terminate)

    for lane in lanes:
        log_path = log_dir / f"{lane.name}.log"
        log_file = log_path.open("a", encoding="utf-8")
        process = subprocess.Popen(
            lane_command(args.server, lane, args.host),
            env=lane_env(lane, args.apc_path),
            stdout=log_file,
            stderr=subprocess.STDOUT,
        )
        processes.append((lane, process))
        print(
            f"started {lane.name} pid={process.pid} "
            f"model={lane.model} url=http://{args.host}:{lane.port}/v1 log={log_path}"
        )

    for lane, process in processes:
        if not wait_ready(args.host, lane, args.wait_timeout):
            print(f"{lane.name} did not become healthy; see logs", file=sys.stderr)
            terminate(signal.SIGTERM, None)
            return 1
        if process.poll() is not None:
            print(f"{lane.name} exited early with code {process.returncode}", file=sys.stderr)
            return process.returncode or 1
        print(f"{lane.name} healthy: http://{args.host}:{lane.port}/v1")

    print_opencode(args)

    while any(process.poll() is None for _, process in processes):
        time.sleep(1.0)
    return max((process.returncode or 0 for _, process in processes), default=0)


def status(args: argparse.Namespace) -> int:
    lanes = selected_lanes(args)
    ok = True
    for lane in lanes:
        health = read_json(health_url(args.host, lane))
        stats = read_json(cache_stats_url(args.host, lane))
        if health is None:
            ok = False
            print(f"{lane.name}: down http://{args.host}:{lane.port}/v1")
            continue
        print(f"{lane.name}: up http://{args.host}:{lane.port}/v1 model={lane.model}")
        if stats is not None:
            matched = stats.get("matched_tokens", 0)
            exact_hits = stats.get("exact_hits", 0)
            disk_gb = round(float(stats.get("disk_bytes", 0)) / 1_000_000_000, 2)
            print(f"  APC matched_tokens={matched} exact_hits={exact_hits} disk_gb={disk_gb}")
    return 0 if ok else 1


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Launch the tested Gemma 4 MLX/APC local inference stack."
    )
    parser.add_argument("command", choices=("serve", "status", "opencode-env"))
    parser.add_argument("--server", default=DEFAULT_SERVER)
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--apc-path", default=DEFAULT_APC_PATH)
    parser.add_argument("--log-dir", default=DEFAULT_LOG_DIR)
    parser.add_argument("--main-port", type=int)
    parser.add_argument("--helper-port", type=int)
    parser.add_argument("--main-context", type=int, default=262144)
    parser.add_argument("--helper-context", type=int, default=131072)
    parser.add_argument("--main-max-tokens", type=int, default=2048)
    parser.add_argument("--helper-max-tokens", type=int, default=1024)
    parser.add_argument("--helper", choices=("helper-e4b", "helper-e2b"), default="helper-e4b")
    parser.add_argument("--main-only", action="store_true")
    parser.add_argument("--helper-only", action="store_true")
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--wait-timeout", type=float, default=180.0)
    return parser


def main() -> int:
    args = build_parser().parse_args()
    if args.main_only and args.helper_only:
        print("--main-only and --helper-only are mutually exclusive", file=sys.stderr)
        return 2
    if args.command == "serve":
        return serve(args)
    if args.command == "status":
        return status(args)
    if args.command == "opencode-env":
        print_opencode(args)
        return 0
    return 2


if __name__ == "__main__":
    raise SystemExit(main())
