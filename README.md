# gator

`gator` is a command-line RSS feed aggregator written in Go. It lets multiple
users register, follow RSS feeds, aggregate posts on a schedule into a Postgres
database, and browse the posts they've collected — all from the terminal.

## Prerequisites

You'll need two things installed to run `gator`:

- **[Go](https://go.dev/doc/install)** (1.26+) — to install the CLI.
- **[PostgreSQL](https://www.postgresql.org/download/)** (15+) — `gator` stores
  users, feeds, and posts in a Postgres database.

You can run Postgres however you like (local install, a managed instance, etc.).
This repo also ships a `docker-compose.yml` if you'd rather spin one up quickly:

```bash
docker compose up -d
```

That starts Postgres on `localhost:5432` with user `postgres`, password
`postgres`, and a database named `gator`.

## Install

Install the `gator` CLI with `go install`:

```bash
go install github.com/OleksandrKirovychDev/gator@latest
```

This compiles a standalone binary and drops it in your `$GOBIN` (usually
`~/go/bin`). Make sure that directory is on your `PATH`, then you can run
`gator` directly from anywhere in your terminal.

## Configuration

`gator` reads its configuration from a JSON file named `.gatorconfig.json` in
your **home directory** (`~/.gatorconfig.json`). Create it with your Postgres
connection string:

```json
{
  "db_url": "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable",
  "current_user_name": ""
}
```

- `db_url` — the Postgres connection string `gator` connects to.
- `current_user_name` — the currently logged-in user. Leave it empty; `gator`
  fills it in for you when you `register` or `login`.

### Database migrations

The schema lives in `sql/schema` as [goose](https://github.com/pressly/goose)
migrations. Apply them once before running the app:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
goose -dir sql/schema postgres "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable" up
```

## Usage

Run a command with:

```bash
gator <command> [args...]
```

A typical first session:

```bash
gator register alice          # create a user and log in as them
gator addfeed "Boot.dev Blog" https://blog.boot.dev/index.xml
gator agg 1m                  # collect posts from feeds every 1 minute (runs until Ctrl+C)
gator browse 5                # show the 5 most recent posts for the current user
```

### Available commands

| Command                | Description                                                       |
| ---------------------- | ----------------------------------------------------------------- |
| `register <name>`      | Create a new user and log in as them.                             |
| `login <name>`         | Switch the active user to an existing one.                        |
| `users`                | List all users (the current one is marked).                       |
| `reset`                | Delete all users (handy during development).                      |
| `addfeed <name> <url>` | Add a feed and automatically follow it.                           |
| `feeds`                | List every feed and who added it.                                 |
| `follow <url>`         | Follow an existing feed by its URL.                               |
| `following`            | List the feeds the current user follows.                          |
| `unfollow <url>`       | Stop following a feed.                                            |
| `agg <duration>`       | Continuously fetch feeds on an interval (e.g. `30s`, `1m`, `1h`). |
| `browse [limit]`       | Show recent posts from followed feeds (default `2`).              |

> **Note:** `addfeed`, `follow`, `following`, `unfollow`, and `browse` require a
> logged-in user — run `register` or `login` first.
