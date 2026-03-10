#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import time
import uuid
from pathlib import Path
from typing import Any

try:
    import jwt
except ImportError:
    print("Error: PyJWT is required. Install it with: pip install PyJWT")
    sys.exit(1)


DEFAULT_BASE_URL = "http://127.0.0.1:8080/api/v1"
DEFAULT_JWT_SECRET = "smap-secret-key-at-least-32-chars-long"
DEFAULT_INTERNAL_KEY = "ingest-service-key"
DEFAULT_KEYWORD = "vinfast vf8 review"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description=(
            "Create one TikTok keyword datasource/target via ingest HTTP APIs, "
            "then trigger manual dispatch via ingest internal API."
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
        "--internal-key",
        default=os.environ.get("INTERNAL_KEY", DEFAULT_INTERNAL_KEY),
        help="X-Internal-Key used by ingest internal routes.",
    )
    parser.add_argument(
        "--keyword",
        default=os.environ.get("TIKTOK_KEYWORD", DEFAULT_KEYWORD),
        help="Keyword value used for the ingest keyword target.",
    )
    parser.add_argument(
        "--artifacts-dir",
        default=os.environ.get("ARTIFACTS_DIR", "./artifacts"),
        help="Directory to save request/response artifacts.",
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

    lines = headers_file.read_text(encoding="utf-8").splitlines()
    for line in reversed(lines):
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


def main() -> int:
    args = parse_args()
    artifacts_dir = ensure_dir(args.artifacts_dir)
    token = generate_bypass_token(args.jwt_secret)
    project_id = str(uuid.uuid4())

    auth_headers = {
        "Authorization": f"Bearer {token}",
    }
    internal_headers = {
        "X-Internal-Key": args.internal_key,
    }

    create_datasource_payload = {
        "project_id": project_id,
        "name": f"Smoke TikTok Search {args.keyword}",
        "source_type": "TIKTOK",
        "source_category": "CRAWL",
        "crawl_mode": "NORMAL",
        "crawl_interval_minutes": 11,
    }

    create_target_payload = {
        "values": [args.keyword],
        "label": f"Keyword {args.keyword}",
        "is_active": True,
        "crawl_interval_minutes": 11,
    }

    print("Step 1: Create datasource via ingest HTTP API")
    status_code, body = run_curl(
        case_name="create_datasource",
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

    print("Step 2: Create keyword target via ingest HTTP API")
    status_code, body = run_curl(
        case_name="create_keyword_target",
        base_url=args.base_url,
        method="POST",
        endpoint=f"/datasources/{data_source_id}/targets/keywords",
        headers=auth_headers,
        payload=create_target_payload,
        artifacts_dir=artifacts_dir,
    )
    data = require_ok(status_code, body, "create keyword target")
    target = data.get("target", {})
    target_id = target.get("id")
    if not target_id:
        raise RuntimeError(f"create keyword target returned no id: {body}")

    print(f"Created target_id: {target_id}")

    print("Step 3: Dispatch target via ingest internal API")
    status_code, body = run_curl(
        case_name="dispatch_target",
        base_url=args.base_url,
        method="POST",
        endpoint=f"/ingest/datasources/{data_source_id}/targets/{target_id}/dispatch",
        headers=internal_headers,
        artifacts_dir=artifacts_dir,
    )
    data = require_ok(status_code, body, "dispatch target")
    task_id = data.get("task_id")
    queue = data.get("queue")
    action = data.get("action")
    if not task_id:
        raise RuntimeError(f"dispatch target returned no task_id: {body}")

    dispatch_artifact = artifacts_dir / f"dispatch_response_{task_id[:8]}.json"
    save_json(dispatch_artifact, body)

    print("")
    print("Dispatch created successfully.")
    print(f"datasource_id:  {data_source_id}")
    print(f"target_id:      {target_id}")
    print(f"task_id:        {task_id}")
    print(f"queue:          {queue}")
    print(f"action:         {action}")
    print(f"dispatch artifact: {dispatch_artifact}")
    print("")
    print(
        "Note: ingest currently dispatches TikTok keyword targets as "
        "`tiktok_tasks/search`, not `full_flow`."
    )
    print("Script stops here. No completion queue consume is performed.")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
