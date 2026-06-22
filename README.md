# go_redis_folder_syncer

Go reimplementation of the Python `folder_syncer` project.

Watches a local directory and syncs every file's content to Redis (base64-encoded).
On startup it restores any cached files from Redis back to disk.

## Why?
If you had a Dockerised service (or similar service) that, for any reason, could not use a volume and needed a folder to persist between runs, this service can help you.

## Requirements

- Go 1.22+
- A running Redis instance

## Build

```bash
go build -o folder-syncer .
```

## Usage

```
folder-syncer start --watch-dir ./data          # background daemon
folder-syncer start --watch-dir ./data --fg     # foreground
folder-syncer status
folder-syncer stop
```

All flags can also be set via environment variables or a `.env` file:

| Flag                | Env var            | Default                      |
|---------------------|--------------------|------------------------------|
| `--watch-dir`       | `SYNC_WATCH_DIR`        | *(required)*                 |
| `--redis-host`      | `SYNC_REDIS_HOST`       | `localhost`                  |
| `--redis-port`      | `SYNC_REDIS_PORT`       | `6379`                       |
| `--redis-key-prefix`| `SYNC_REDIS_KEY_PREFIX` | `file_cache:`                |
| `--redis-password`  | `SYNC_REDIS_PASSWORD`   | ``                           |
| `--redis-db`        | `SYNC_REDIS_DB`         | `0`                          |
| `--pid-file`        | `SYNC_PID_FILE`         | `/tmp/folder_syncer.pid`     |
| `--log-file`        | `SYNC_LOG_FILE`         | `/tmp/folder_syncer.log`     |

## Test

```bash
go test ./...
```

## Project layout

```
.
├── main.go
├── cmd/            # cobra CLI commands (start / stop / status)
├── internal/
│   ├── app/        # dependency wiring and core run loop
│   ├── config/     # settings loaded from env / .env
│   ├── daemon/     # PID file helpers, daemonization
│   └── sync/       # SyncService + fsnotify Watcher
└── .env.example
```
