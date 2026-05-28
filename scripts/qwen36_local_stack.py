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
DEFAULT_APC_PATH = "/private/tmp/mlx-vlm-apc-qwen36"
DEFAULT_LOG_DIR = "/private/tmp/core-agent-qwen36-stack"


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
    "coding27": Lane(
        name="coding27",
        role="main",
        model="mlx-community/Qwen3.6-27B-4bit",
        port=8003,
        max_kv_size=262144,
        max_tokens=4096,
        apc_blocks=24000,
        apc_disk_gb=48,
    ),
    "coding27-mxfp8": Lane(
        name="coding27-mxfp8",
        role="main",
        model="mlx-community/Qwen3.6-27B-mxfp8",
        port=8006,
        max_kv_size=262144,
        max_tokens=4096,
        apc_blocks=24000,
        apc_disk_gb=48,
    ),
    "moe35": Lane(
        name="moe35",
        role="helper",
        model="mlx-community/Qwen3.6-35B-A3B-4bit",
        port=8008,
        max_kv_size=262144,
        max_tokens=2048,
        apc_blocks=24000,
        apc_disk_gb=48,
    ),
}


def lane_env(lane: Lane, args: argparse.Namespace) -> dict[str, str]:
    env = os.environ.copy()
    if args.mode != "apc":
        env["APC_ENABLED"] = "0"
        return env
    env.update(
        {
            "APC_ENABLED": "1",
            "APC_NUM_BLOCKS": str(lane.apc_blocks),
            "APC_BLOCK_SIZE": "16",
            "APC_LAYER_MAJOR_MEMORY_MIN_TOKENS": "50000",
            "APC_DISK_PATH": args.apc_path,
            "APC_DISK_MAX_GB": str(lane.apc_disk_gb),
            "APC_DISK_SHARD_MAX_BLOCKS": "256",
        }
    )
    return env


def lane_command(server: str, lane: Lane, args: argparse.Namespace) -> list[str]:
    command = [
        server,
        "--host",
        args.host,
        "--port",
        str(lane.port),
        "--model",
        lane.model,
        "--max-kv-size",
        str(lane.max_kv_size),
        "--max-tokens",
        str(lane.max_tokens),
        "--prefill-step-size",
        str(args.prefill_step_size),
    ]
    if args.mode == "turboquant":
        command.extend(
            [
                "--kv-bits",
                str(args.kv_bits),
                "--kv-quant-scheme",
                "turboquant",
                "--quantized-kv-start",
                str(args.quantized_kv_start),
            ]
        )
    return command


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
    base = LANES[args.lane]
    lane = replace(
        base,
        port=args.port if args.port is not None else base.port,
        max_kv_size=args.context if args.context is not None else base.max_kv_size,
        max_tokens=args.max_tokens if args.max_tokens is not None else base.max_tokens,
    )
    return [lane]


def print_commands(args: argparse.Namespace, lanes: Iterable[Lane]) -> None:
    for lane in lanes:
        env = lane_env(lane, args)
        if args.mode == "apc":
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
        else:
            env_prefix = "APC_ENABLED=0"
        command = " ".join(lane_command(args.server, lane, args))
        print(f"{lane.name}: {env_prefix} {command}")


def print_env(args: argparse.Namespace) -> None:
    lane = selected_lanes(args)[0]
    profile = lane.name.replace("-", "_").upper()
    print("# CoreAgent/OpenCode profile overrides for this Qwen3.6 lane")
    print(f"export CORE_OPENCODE_QWEN36_{profile}_BASE_URL=http://{args.host}:{lane.port}/v1")
    print(f"export CORE_OPENCODE_QWEN36_{profile}_MODEL={lane.model}")
    print()
    if lane.name == "coding27":
        print('scripts/local-agent.sh --profile qwen36 "summarise the current coding task"')
    elif lane.name == "coding27-mxfp8":
        print('scripts/local-agent.sh --profile qwen36-mxfp8 "summarise the current coding task"')
    else:
        print('scripts/local-agent.sh --profile qwen36-moe "summarise the current coding task"')


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
        log_path = log_dir / f"{lane.name}-{args.mode}.log"
        log_file = log_path.open("a", encoding="utf-8")
        process = subprocess.Popen(
            lane_command(args.server, lane, args),
            env=lane_env(lane, args),
            stdout=log_file,
            stderr=subprocess.STDOUT,
        )
        processes.append((lane, process))
        print(
            f"started {lane.name} pid={process.pid} mode={args.mode} "
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

    print_env(args)

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
        description="Launch Qwen3.6 MLX local inference lanes for CoreAgent."
    )
    parser.add_argument("command", choices=("serve", "status", "opencode-env"))
    parser.add_argument("--server", default=DEFAULT_SERVER)
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--apc-path", default=DEFAULT_APC_PATH)
    parser.add_argument("--log-dir", default=DEFAULT_LOG_DIR)
    parser.add_argument("--lane", choices=tuple(LANES), default="coding27")
    parser.add_argument("--mode", choices=("apc", "turboquant"), default="apc")
    parser.add_argument("--port", type=int)
    parser.add_argument("--context", type=int)
    parser.add_argument("--max-tokens", type=int)
    parser.add_argument("--prefill-step-size", type=int, default=2048)
    parser.add_argument("--kv-bits", type=float, default=3.5)
    parser.add_argument("--quantized-kv-start", type=int, default=4096)
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--wait-timeout", type=float, default=240.0)
    return parser


def main() -> int:
    args = build_parser().parse_args()
    if args.command == "serve":
        return serve(args)
    if args.command == "status":
        return status(args)
    if args.command == "opencode-env":
        print_env(args)
        return 0
    return 2


if __name__ == "__main__":
    raise SystemExit(main())
