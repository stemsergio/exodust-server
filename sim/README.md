# sim

The game as a pure function: `Tick(state, inputs) -> state`.

Rules:
- Imports only the standard library and `physics`. Never network, DB or `cmd/`.
- Fixed-point integer math. No floats in simulation state.
- Deterministic: no randomness without a seed in state, no time.Now(), no unsorted map iteration.
- ECS: entity = id, component = flat data, system = function.

Owns everything that is Exodust-specific:
- Input → movement intent, skills, attacks.
- Ghost and immunity rules, given to `physics` as its pair filter.
- Impact damage, computed from the contacts and impulses `physics` reports.
- Kill attribution, health, death, respawn.

Order per tick:
1. Apply inputs (movement intent, skill casts)
2. `physics.Step` (integrate, collide, resolve, sweep projectiles, raycasts)
3. Read contacts and hits → damage, deaths, ghost relations
