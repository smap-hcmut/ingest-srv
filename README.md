# ingest-srv

SMAP Ingest Service boilerplate.

## Run

```bash
make run-api
make run-consumer
make run-scheduler
```

## Config

Configuration file path:

- `config/ingest-config.yaml`
- `config/ingest-config.example.yaml`

## Endpoints

- `GET /health`
- `GET /ready`
- `GET /live`
- `GET /api/v1/ingest/ping`
