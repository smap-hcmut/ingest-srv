#!/usr/bin/env python3
"""Publish a CrawlRequest message to the collector inbound exchange (hardcoded payload).

The dispatcher consumer (`cmd/consumer`) listens on exchange `collector.inbound`
with routing key pattern `crawler.#` and will fan-out tasks to worker queues.
"""

import json
import os
import uuid
from datetime import datetime, timezone

import pika

# --- Hardcoded message config ---
PLATFORM = "youtube"
TASK_TYPE = "research_keyword"
PAYLOAD = {"keyword": "python"}
TIME_RANGE = 7
ENV_FILE = ".env"  # used only to read RABBITMQ_* if not in environment
ROUTING_KEY = "crawler.youtube.research_keyword"


def load_env_file(path: str) -> dict:
    """Minimal .env loader (no external dependency)."""
    env = {}
    if not path or not os.path.exists(path):
        return env

    with open(path, "r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, value = line.split("=", 1)
            env[key.strip()] = value.strip().strip('"').strip("'")
    return env


def resolve_rabbitmq_url(env_from_file: dict) -> str:
    """Resolve RabbitMQ URL from env vars > .env > host parts."""
    env = os.environ

    url = env.get("RABBITMQ_URL") or env_from_file.get("RABBITMQ_URL")
    if url:
        return url

    host = env.get("RABBITMQ_HOST") or env_from_file.get("RABBITMQ_HOST") or "localhost"
    port = env.get("RABBITMQ_PORT") or env_from_file.get("RABBITMQ_PORT") or "5672"
    user = env.get("RABBITMQ_USER") or env_from_file.get("RABBITMQ_USER") or "guest"
    password = env.get("RABBITMQ_PASSWORD") or env_from_file.get("RABBITMQ_PASSWORD") or "guest"
    vhost = env.get("RABBITMQ_VHOST") or env_from_file.get("RABBITMQ_VHOST") or "/"
    vhost = vhost[1:] if vhost.startswith("/") else vhost
    return f"amqp://{user}:{password}@{host}:{port}/{vhost}"


def main():
    env_from_file = load_env_file(ENV_FILE)
    rabbit_url = resolve_rabbitmq_url(env_from_file)

    if not rabbit_url:
        raise SystemExit("RabbitMQ URL not found. Set RABBITMQ_URL (env or .env).")

    payload = dict(PAYLOAD)
    if TIME_RANGE is not None and isinstance(payload, dict) and "time_range" not in payload:
        payload["time_range"] = TIME_RANGE

    job_id = f"job-{uuid.uuid4().hex[:8]}"
    body = {
        "job_id": job_id,
        "task_type": TASK_TYPE,
        "payload": payload,
        "emitted_at": datetime.now(timezone.utc).isoformat(),
    }
    if TIME_RANGE is not None:
        body["time_range"] = TIME_RANGE

    params = pika.URLParameters(rabbit_url)
    connection = pika.BlockingConnection(params)
    channel = connection.channel()

    channel.exchange_declare(exchange="collector.inbound", exchange_type="topic", durable=True)
    channel.queue_declare(queue="collector.inbound.queue", durable=True)
    channel.queue_bind(queue="collector.inbound.queue", exchange="collector.inbound", routing_key="crawler.#")

    channel.basic_publish(
        exchange="collector.inbound",
        routing_key=ROUTING_KEY,
        body=json.dumps(body),
        properties=pika.BasicProperties(content_type="application/json", delivery_mode=2),
    )

    print(f"Published job_id={job_id} task_type={TASK_TYPE} to {ROUTING_KEY}")
    connection.close()


if __name__ == "__main__":
    main()
