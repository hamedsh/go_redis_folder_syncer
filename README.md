# go_redis_folder_syncer

Go reimplementation of the Python `folder_syncer` project.

Watches a local directory and syncs every file's content to Redis (base64-encoded).
On startup it restores any cached files from Redis back to disk.

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
| `--watch-dir`       | `WATCH_DIR`        | *(required)*                 |
| `--redis-host`      | `REDIS_HOST`       | `localhost`                  |
| `--redis-port`      | `REDIS_PORT`       | `6379`                       |
| `--redis-key-prefix`| `REDIS_KEY_PREFIX` | `file_cache:`                |
| `--pid-file`        | `PID_FILE`         | `/tmp/folder_syncer.pid`     |
| `--log-file`        | `LOG_FILE`         | `/tmp/folder_syncer.log`     |

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
