# physics

Generic, deterministic 3D physics for the server. **Knows nothing about Exodust**:
no players, no damage, no ghosts. Could be reused in another game or published as-is.

Rules:
- Imports only the standard library. Never `sim`, `cmd/`, network or DB.
- Fixed-point integers (Q48.16), rounding toward zero. No floats in state.
- Deterministic: same bodies and same inputs give the same result on any machine.

Hooks the game provides:
- `PairFilter(a, b) bool` — whether two bodies may collide (sim plugs in the ghost rule).
- Contact/hit report out of each step — `sim` turns impulses into damage.

Pipeline per step:
1. Integrate motion (gravity, damping, speed caps)
2. Broadphase (hashed spatial grid) → candidate pairs
3. Pair filter → collidable pairs
4. Narrowphase → contacts
5. Resolve contacts (impulses, friction, position correction)
6. Swept projectiles and raycasts (first hit)

Spec, measured costs and milestones: `../PHYSICS.md`.
