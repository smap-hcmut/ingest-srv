#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import time
import uuid
from dataclasses import dataclass
from pathlib import Path
from typing import Any

try:
    import jwt
except ImportError:
    print("Error: PyJWT is required. Install it with: pip install PyJWT")
    sys.exit(1)

try:
    import psycopg
except ImportError:
    print("Error: psycopg is required. Install it with: pip install psycopg[binary]")
    sys.exit(1)


DEFAULT_BASE_URL = "http://127.0.0.1:8080/api/v1"
DEFAULT_JWT_SECRET = "smap-secret-key-at-least-32-chars-long"
DEFAULT_ARTIFACTS_DIR = "./artifacts"
DEFAULT_OFFSETS = "-15,-5,-1,5,15"

DEFAULT_DB_HOST = os.environ.get("POSTGRES_HOST", "172.16.19.10")
DEFAULT_DB_PORT = int(os.environ.get("POSTGRES_PORT", "5432"))
DEFAULT_DB_USER = os.environ.get("POSTGRES_USER", "ingest_master")
DEFAULT_DB_PASSWORD = os.environ.get("POSTGRES_PASSWORD", "ingest_master_pwd")
DEFAULT_DB_NAME = os.environ.get("POSTGRES_DBNAME", "smap")
DEFAULT_DB_SCHEMA = os.environ.get("POSTGRES_SCHEMA", "schema_ingest")


@dataclass
class SeededTarget:
    target_id: str
    keyword: str
    offset_minutes: int
    next_crawl_at: str


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description=(
            "Create one TikTok crawl datasource and seed 3-5 keyword targets with "
            "different next_crawl_at values for scheduler testing."
        ),
    )
    parser.add_argument(
        "--base-url",
        default=os.environ.get("BASE_URL", DEFAULT_BASE_URL),
        help="Ingest API base URL, e.g. http://127.0.0.1:8080/api/v1",
    )
    parser.add_argument(
        "--jwt-secret",
        default=os.environ.get("JWT_SECRET_KEY", DEFAULT_JWT_SECRET),
        help="JWT secret used by ingest auth middleware.",
    )
    parser.add_argument(
        "--artifacts-dir",
        default=os.environ.get("ARTIFACTS_DIR", DEFAULT_ARTIFACTS_DIR),
        help="Directory to save request/response artifacts.",
    )
    parser.add_argument(
        "--project-id",
        default=os.environ.get("PROJECT_ID", str(uuid.uuid4())),
        help="Project ID used for the seeded datasource.",
    )
    parser.add_argument(
        "--offsets",
        default=os.environ.get("NEXT_CRAWL_OFFSETS", DEFAULT_OFFSETS),
        help="Comma-separated minute offsets from now for next_crawl_at, e.g. -15,-5,5",
    )
    parser.add_argument(
        "--db-host",
        default=DEFAULT_DB_HOST,
        help="PostgreSQL host.",
    )
    parser.add_argument(
        "--db-port",
        type=int,
        default=DEFAULT_DB_PORT,
        help="PostgreSQL port.",
    )
    parser.add_argument(
        "--db-user",
        default=DEFAULT_DB_USER,
        help="PostgreSQL user.",
    )
    parser.add_argument(
        "--db-password",
        default=DEFAULT_DB_PASSWORD,
        help="PostgreSQL password.",
    )
    parser.add_argument(
        "--db-name",
        default=DEFAULT_DB_NAME,
        help="PostgreSQL database name.",
    )
    parser.add_argument(
        "--db-schema",
        default=DEFAULT_DB_SCHEMA,
        help="PostgreSQL schema name.",
    )
    return parser.parse_args()


def ensure_dir(path: str) -> Path:
    dir_path = Path(path)
    dir_path.mkdir(parents=True, exist_ok=True)
    return dir_path


def save_json(path: Path, payload: dict[str, Any]) -> None:
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")


def now_unix() -> int:
    return int(time.time())


def generate_bypass_token(secret_key: str) -> str:
    payload = {
        "email": "qa@test.com",
        "role": "admin",
        "groups": [],
        "iss": "smap",
        "sub": str(uuid.uuid4()),
        "exp": now_unix() + 3600,
        "iat": now_unix(),
        "jti": str(uuid.uuid4()),
    }
    return jwt.encode(payload, secret_key, algorithm="HS256")


def parse_status_code(headers_file: Path) -> int:
    if not headers_file.exists():
        return 0

    for line in reversed(headers_file.read_text(encoding="utf-8").splitlines()):
        if line.startswith("HTTP/"):
            parts = line.split()
            if len(parts) >= 2 and parts[1].isdigit():
                return int(parts[1])
    return 0


def run_curl(
    *,
    case_name: str,
    base_url: str,
    method: str,
    endpoint: str,
    artifacts_dir: Path,
    headers: dict[str, str] | None = None,
    payload: dict[str, Any] | None = None,
) -> tuple[int, Any]:
    url = f"{base_url}{endpoint}"
    headers_file = artifacts_dir / f"{case_name}_headers.txt"
    body_file = artifacts_dir / f"{case_name}_body.json"
    request_file = artifacts_dir / f"{case_name}_request.json"

    cmd = ["curl", "-sS", "-D", str(headers_file), "-o", str(body_file), "-X", method, url]

    for key, value in (headers or {}).items():
        cmd.extend(["-H", f"{key}: {value}"])

    if payload is not None:
        save_json(request_file, payload)
        cmd.extend(["-H", "Content-Type: application/json"])
        cmd.extend(["-d", json.dumps(payload, ensure_ascii=False)])

    try:
        subprocess.run(cmd, check=True, capture_output=True)
    except subprocess.CalledProcessError as exc:
        stderr = exc.stderr.decode("utf-8", errors="ignore")
        raise RuntimeError(f"curl failed for {endpoint}: {stderr}") from exc

    status_code = parse_status_code(headers_file)
    raw_body = body_file.read_text(encoding="utf-8") if body_file.exists() else ""
    try:
        body = json.loads(raw_body) if raw_body else None
    except json.JSONDecodeError:
        body = raw_body

    return status_code, body


def require_ok(status_code: int, body: Any, context: str) -> dict[str, Any]:
    if status_code != 200 or not isinstance(body, dict):
        raise RuntimeError(f"{context} failed: status={status_code}, body={body}")

    data = body.get("data")
    if not isinstance(data, dict):
        raise RuntimeError(f"{context} returned unexpected body: {body}")
    return data


def parse_offsets(raw_offsets: str) -> list[int]:
    offsets = []
    for item in raw_offsets.split(","):
        value = item.strip()
        if not value:
            continue
        offsets.append(int(value))
    if len(offsets) < 3 or len(offsets) > 5:
        raise ValueError("offsets must contain between 3 and 5 values")
    return offsets


def activate_datasource_and_set_targets(
    *,
    args: argparse.Namespace,
    data_source_id: str,
    target_offsets: list[tuple[str, int]],
) -> list[SeededTarget]:
    dsn = (
        f"host={args.db_host} port={args.db_port} dbname={args.db_name} "
        f"user={args.db_user} password={args.db_password}"
    )
    seeded_targets: list[SeededTarget] = []

    with psycopg.connect(dsn) as conn:
        with conn.cursor() as cur:
            cur.execute(f"SET search_path TO {args.db_schema}")

            cur.execute(
                """
                UPDATE data_sources
                SET status = 'ACTIVE',
                    activated_at = NOW(),
                    updated_at = NOW()
                WHERE id = %s
                """,
                (data_source_id,),
            )

            for target_id, offset_minutes in target_offsets:
                cur.execute(
                    """
                    UPDATE crawl_targets
                    SET next_crawl_at = NOW() + (%s * INTERVAL '1 minute'),
                        updated_at = NOW()
                    WHERE id = %s
                    RETURNING next_crawl_at
                    """,
                    (offset_minutes, target_id),
                )
                row = cur.fetchone()
                if row is None:
                    raise RuntimeError(f"failed to update next_crawl_at for target_id={target_id}")

                seeded_targets.append(
                    SeededTarget(
                        target_id=target_id,
                        keyword="",
                        offset_minutes=offset_minutes,
                        next_crawl_at=row[0].isoformat(),
                    )
                )

        conn.commit()

    return seeded_targets


def main() -> int:
    args = parse_args()
    offsets = parse_offsets(args.offsets)
    artifacts_dir = ensure_dir(args.artifacts_dir)
    token = generate_bypass_token(args.jwt_secret)
    auth_headers = {"Authorization": f"Bearer {token}"}

    suffix = uuid.uuid4().hex[:8]
    datasource_name = f"Scheduler Seed TikTok {suffix}"

    create_datasource_payload = {
        "project_id": args.project_id,
        "name": datasource_name,
        "source_type": "TIKTOK",
        "source_category": "CRAWL",
        "crawl_mode": "NORMAL",
        "crawl_interval_minutes": 11,
    }

    print("Step 1: Create datasource")
    status_code, body = run_curl(
        case_name="seed_scheduler_datasource",
        base_url=args.base_url,
        method="POST",
        endpoint="/datasources",
        headers=auth_headers,
        payload=create_datasource_payload,
        artifacts_dir=artifacts_dir,
    )
    data = require_ok(status_code, body, "create datasource")
    data_source = data.get("data_source", {})
    data_source_id = data_source.get("id")
    if not data_source_id:
        raise RuntimeError(f"create datasource returned no id: {body}")

    print(f"Created datasource_id: {data_source_id}")

    created_targets: list[tuple[str, str, int]] = []
    print("Step 2: Create keyword targets")
    for index, offset in enumerate(offsets, start=1):
        keyword = f"scheduler seed keyword {suffix}-{index}"
        create_target_payload = {
            "values": [keyword],
            "label": f"Seed Target {index}",
            "is_active": True,
            "priority": max(0, len(offsets) - index),
            "crawl_interval_minutes": 11,
        }
        status_code, body = run_curl(
            case_name=f"seed_scheduler_target_{index}",
            base_url=args.base_url,
            method="POST",
            endpoint=f"/datasources/{data_source_id}/targets/keywords",
            headers=auth_headers,
            payload=create_target_payload,
            artifacts_dir=artifacts_dir,
        )
        data = require_ok(status_code, body, f"create target {index}")
        target = data.get("target", {})
        target_id = target.get("id")
        if not target_id:
            raise RuntimeError(f"create target {index} returned no id: {body}")
        created_targets.append((target_id, keyword, offset))
        print(f"  - target_id={target_id} keyword={keyword} offset={offset}m")

    print("Step 3: Activate datasource and set next_crawl_at offsets in DB")
    seeded_targets = activate_datasource_and_set_targets(
        args=args,
        data_source_id=data_source_id,
        target_offsets=[(target_id, offset) for target_id, _, offset in created_targets],
    )

    target_summaries = []
    for seeded, (_, keyword, offset) in zip(seeded_targets, created_targets):
        seeded.keyword = keyword
        seeded.offset_minutes = offset
        target_summaries.append(
            {
                "target_id": seeded.target_id,
                "keyword": seeded.keyword,
                "offset_minutes": seeded.offset_minutes,
                "next_crawl_at": seeded.next_crawl_at,
            }
        )

    summary = {
        "project_id": args.project_id,
        "datasource_id": data_source_id,
        "datasource_name": datasource_name,
        "scheduler_heartbeat_limit": len(offsets),
        "targets": target_summaries,
    }
    summary_path = artifacts_dir / f"seed_scheduler_targets_{suffix}.json"
    save_json(summary_path, summary)

    print("")
    print("Seed completed.")
    print(f"datasource_id: {data_source_id}")
    print(f"summary:       {summary_path}")
    print("")
    print("Seeded targets:")
    for item in target_summaries:
        print(
            f"- {item['target_id']} | offset={item['offset_minutes']}m | "
            f"next_crawl_at={item['next_crawl_at']} | keyword={item['keyword']}"
        )

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
