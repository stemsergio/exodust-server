# cmd/server

Entry point (`main.go`). Wraps `sim`:
- HTTP server with a WebSocket endpoint.
- One goroutine reads each connection and queues inputs.
- A single loop goroutine owns the world state: drains inputs, calls `sim.Tick`, broadcasts snapshots.

Only the loop goroutine touches world state. That is the main defense against race conditions.
