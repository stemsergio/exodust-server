# Server (Go)

Authoritative game server. One Go module, one binary per `cmd/` folder.

```
Server/
├── go.mod
├── physics/      generic engine: bodies, shapes, collisions. Imports nothing from the project
├── sim/          the game: ECS, combat, ghost rule. Imports physics
└── cmd/
    └── server/   main: WebSocket endpoint, fixed-tick loop, snapshot broadcast. Imports sim
```

Imports flow one way: `cmd/*` → `sim` → `physics`. Go forbids import cycles, so a
violation does not compile. Later binaries (bot, replay) go in `cmd/` next to `server/`.
Each package can become its own repo later by moving the folder.

- Tick rate: 30 Hz (configurable). All gameplay durations are expressed in ticks.
- Players authenticate with the token issued by exodustWeb; the server only verifies it.
- No database access. Persistent results go to exodustWeb over HTTP.
- Config and secrets from environment variables (`Secrets/server.env` locally).

Physics spec, costs and progress: `PHYSICS.md`.

Run tests: `go test -race ./...`
