# Exodust — Physics spec

What the server simulates, in what order to build it, what it costs, and what we skip.
Lives next to the code it describes: the generic engine in `Server/physics`, the game-specific
parts (ghost filter, impact damage, kill attribution) in `Server/sim`. Update the status column
as things land.

Costs marked **measured** come from a throwaway prototype (Go 1.24, one core of an
Intel Xeon @ 2.8 GHz cloud vCPU, similar to a cheap VPS). Treat them as order of magnitude:
your machine will differ by ±2×. Costs marked **est.** are estimates.

Tags used below: **Off** = something in the original list that is not quite right.
**Missing** = not in the original list but needed. **Skip** = not worth doing (now or ever).

Last updated: 2026-10-08

---

## 0. Conventions — decide once, never change silently

| Item | Decision |
|---|---|
| Axes | Right-handed, **Y up** (same as Three.js and glTF). Gravity points to −Y. |
| Units in design data | SI: m, s, kg, m/s, m/s², N, N·s. Designers write "walk speed 6 m/s". |
| Units inside `sim` | Position in m. Velocity in **m per tick**. Acceleration in **m per tick²**. Time in ticks. Converted once when data loads (6 m/s → 0.2 m/tick). |
| Tick | 30 Hz fixed, dt = 1/30 s. Budget: 33.3 ms per tick for **all** systems, not only physics. |
| Numbers | Fixed point **Q48.16** in `int64`: 1 m = 65 536. Resolution 0.015 mm. |
| Rounding | Toward zero, only through helper functions. Never raw `>>` or `/` on signed values in gameplay code. |
| Integrator | Semi-implicit Euler: `v += a`, then `x += v`. |

Per-tick units mean every tick is just two additions: no multiply by dt, no rounding.

### Fixed-point facts (measured)

- Gravity 9.81 m/s² → 714 raw per tick² → 9.8053 m/s² (−0.05 % error). Fine.
- A 10 m drop takes 43 ticks (1.433 s) vs 1.428 s analytic. The 1-tick gap is the integrator, not fixed point.
- Velocity resolution: 1 raw unit per tick = 0.46 mm/s.
- Overflow: a product of two Q16 values must stay below 2⁶³. A dot product of **absolute** positions
  overflows beyond ~26 km. Rule: always multiply **relative** vectors (`b − a`), keep each instance within ±10 km.
- Some values need more precision than Q16: inverse mass (1/1000 kg → 66 raw, 0.7 % error) and per-tick
  factors such as damping. Give them their own format (e.g. Q32) when it matters.

### The classic fixed-point bug (measured)

`(v * 0.98) >> 16` rounds toward −∞. Positive velocities decay to 0, but negative ones get stuck at
−49 raw units forever: every "resting" body drifts **22 mm/s toward −X/−Z**, permanently.
Rounding toward zero makes both decay to 0. That is why rounding goes through helpers.

### Square roots

Needed for lengths, normalization, quadratic drag, raycasts.

| Method | Cost (measured) | Notes |
|---|---|---|
| Integer Newton | 56 ns | Exact `floor(√n)`, zero floats. Start here. |
| Float guess + integer correction | 3 ns | `math.Sqrt` as a first guess, then fixed with integer checks. The result is still the exact integer, so determinism holds. Only allowed exception to "no floats"; input must be < 2⁶². |

Verified: both return identical results on 2 million random inputs. Switch only if profiling says so.

---

## 1. Milestones and status

Build in this order. Each milestone ends with a test that proves it.
Size is rough, for one person learning Go: **S** an evening or two, **M** a weekend, **L** a week or more.

| # | Milestone | Includes | Done when | Size | Status |
|---|---|---|---|---|---|
| M0 | Math library | Q16 type, `Mul`/`Div` with symmetric rounding, `isqrt`, vec3, length, normalize, sin/cos table | +v and −v both decay to 0 under damping; `isqrt` exact on 2M random values | M | TODO |
| M1 | Motion | Integration, gravity, gravity scale per entity, damping, speed caps | 10 m drop ≈ 43 ticks; 10 000 ticks with the same inputs give the same state hash on Windows and Linux | S | TODO |
| M2 | Ground | Flat plane → heightmap, grounded state, jump | Jump height matches `v0²/2g` within one tick | M | TODO |
| M3 | Collisions | Hashed grid, capsule/sphere narrowphase, contact solver (restitution, friction, position correction) | Heavier capsule pushes lighter one in proportion to mass; 264-character crowd at rest jitters < 1 cm | L | TODO |
| M4 | Character movement | Accelerate toward target, air control, input cap vs external cap | A 20 m/s knockback survives holding "forward" | M | TODO |
| M5 | Collision rules | Collision matrix, ghost and immunity filter | Ghost pair passes through each other both ways; immunity ends at exactly kill tick + 60 | S | TODO |
| M6 | Attacks | Beam raycast with grid traversal, swept projectiles, first hit | A 300 m/s projectile never passes through a capsule; a beam hits the nearer of two aligned capsules | M | TODO |
| M7 | Impact damage | Δv-based damage, fall damage, wall slam, kill attribution | 3 m fall = 0 damage, 10 m fall > 0; resting contact = 0 | S | TODO |
| M8 | Static world | Static boxes, slopes, steps, ground snapping | Walk up 0.3 m steps; can't walk up > 45°; no "flying off" downhill | L | TODO |
| M9 | Volumes | Drag zones (foliage, mud, water), optional buoyancy | Body inside a zone reaches the zone's terminal speed | S | TODO |
| M10 | Performance pass | Early-outs, sleeping, profiling | Brawl tick < 2 ms on the target VPS | M | TODO |
| — | Later | Rotation for props, springs/tethers, wind zones with cone shelters, moving platforms | — | — | — |

---

## 2. Topics

### 2.1 Velocity, acceleration, gravity

- **Velocity** is a vector (direction + magnitude) in m/s. Speed is its length `|v|`.
- **Off:** "acc = vel++". The idea is right, the notation isn't: acceleration is the rate of change of
  velocity, `a = Δv/Δt`, in m/s². In code, each tick: `v += a` (per-tick units). `++` adds exactly 1.
- **Off:** gravity is an **acceleration**, not a force. The force is weight: `m·g`. Without drag,
  a 50 kg and a 150 kg character fall exactly the same. Weight does not make things fall faster;
  heaviness shows up in collisions and knockback (2.2).
- Order matters: `v += a` first, then `x += v` (semi-implicit Euler). The other order gains energy
  every tick and makes springs explode.
- **Missing:** gravity tuned for feel. Real g makes jumps feel floaty in most games. Common trick:
  stronger gravity while falling than while rising (1.5–2.5×). To jump h meters: `v0 = √(2gh)`
  (1 m jump at 9.81 → 4.43 m/s).
- **Missing:** gravity scale per entity (0 for beams and some projectiles, 1 for bodies).

Cost (measured): **4 ns per body per tick** for gravity + damping + ground + cap check. 1 000 bodies = 4 µs.

### 2.2 Mass, momentum, impulse, knockback

- Mass `m` (kg) is not weight (N). What you called "peso × velocidad" is **momentum**: `p = m·v`.
- An **impulse** is a change of momentum: `J = Δp = F·Δt` (N·s). Knockback is an impulse:
  `Δv = J / m`. The same hit moves a 60 kg character twice as much as a 120 kg one. This is what makes
  weight matter in combat.
- **Missing:** store **inverse mass** `1/m`. Static world gets `1/m = 0` (infinite mass), so walls need no
  special case, and the solver multiplies instead of dividing.
- Pushing between characters splits automatically by inverse mass.
- Energy `E = ½ m v²` is for damage (2.10); momentum is for pushing and knockback.

### 2.3 Speed caps and the knockback trap

- **Missing:** two different caps.
  - **Input cap**: how fast your own legs take you (e.g. 6 m/s). Input accelerates toward a target
    velocity with a maximum acceleration; it never adds beyond the cap.
  - **External cap**: safety limit for everything (e.g. 40 m/s), against tunneling and absurd launches.
- **The trap:** clamping total velocity to walk speed every tick deletes knockback. Input may only add
  speed up to its cap; it never removes speed that came from outside (ground deceleration does that).
- Terminal velocity in reality comes from air drag (a human falling: ~55 m/s). The external cap does
  that job; no real air drag needed.

Cost (measured): free on the normal path; ~100 ns only for a body over the cap that tick.

### 2.4 Damping

- `v *= (1 − c)` each tick. Not a real force: a cheap stand-in for all losses. Also stabilizes the solver.
- Cost: two multiplications. Free.
- The rounding bug in section 0 lives exactly here.

### 2.5 Drag ("volumetric friction")

- **Off:** surface friction and drag are different things. Friction happens at contacts. Drag happens
  when moving through a medium (air, water, mud, bushes).
- **Linear drag** `F = −b·v`: good for mud, bushes, slow zones. Same cost as damping.
- **Quadratic drag** `F = −½ ρ C_d A |v| v`: real air and water at speed. Needs `|v|` (a square root)
  per body per tick. Gives a natural terminal speed `√(2mg / (ρ C_d A))`.
- **Your question about trees:** in reality, wind slows inside a forest because every branch and leaf
  adds drag (modeled as drag per unit volume × leaf area density). No game simulates leaves. Do it
  with **volumes**: a forest, bush, water or mud zone has a drag coefficient and, when wind arrives,
  a shelter factor. Your cone-shaped wind shelters are the same idea.
- **Skip:** air drag on characters outside special zones.

Cost (measured parts): linear 2–4 ns per body; quadratic +3 ns (fast sqrt) or +56 ns (Newton) per body
inside a volume. Only bodies inside volumes pay.

### 2.6 Surface friction and ground movement

- Coulomb friction: friction force ≤ `μ·N` (N = normal force). Static μ (to start sliding) is higher
  than kinetic μ. In an impulse solver: friction impulse clamped to `μ × normal impulse`.
- Combining two materials: `μ = √(μ1·μ2)` is the usual choice.
- **Skip:** moving characters with real friction. Use tuned ground acceleration/deceleration, and lower
  them on ice or mud. Real friction is for props, knocked-back bodies sliding, and stunned characters.

Cost: part of the contact solver (2.9).

### 2.7 Elastics — two different things

- **Restitution** `e` (bounce): 0 = no bounce (clay), 1 = perfect bounce. Characters 0–0.1, balls
  0.6–0.9. Needs a minimum impact speed below which `e = 0`, or resting bodies jitter.
- **Springs** (Hooke): `F = −k·x − c·v`. Ropes, tethers, trampolines, grapples.
  Stiff springs explode with a fixed tick. Stability for semi-implicit Euler: `ω·dt < 2` with
  `ω = √(k/m)`. At 30 Hz: `k < 3 600 × m` → for 80 kg, `k < 288 000 N/m`. Stay well below
  (10–25 % of that).

Cost (est.): springs 20–80 ns each (they need a length). Rare; free in practice.

### 2.8 Collision detection

**Shapes**

| Thing | Shape | Why |
|---|---|---|
| Characters | Vertical capsule, r 0.4 m, h 1.8 m | Smooth, cheap, no corners that snag |
| Projectiles | Sphere, swept | Cheap; sweeping prevents tunneling |
| Props | Sphere / capsule first | Boxes need rotation (2.14) |
| Terrain | Heightmap | Height under a point is one cheap lookup |
| Structures | Static boxes / capsules | Primitives, never render meshes |

**Missing:** collision shapes are not render meshes. In Blender, model simplified collision
primitives (e.g. objects named `COL_*`); export only those for the server. The server never loads
render meshes or textures.

**Broadphase** (who might touch)

- Hashed 2D grid on X/Z, 2 m cells; each body checks its 3×3 neighbor cells. Characters rarely
  stack vertically, so 2D is enough to start.
- Measured candidate pairs → real contacts: typical **1 385 → 113**; brawl (264 characters packed in a
  ~7 m radius) **9 554 → 742**.
- Bodies can't overlap much, so even the worst brawl caps density at ~7–8 characters per cell.
  That is why N² never happens.

**Collision matrix**

| | Character | NPC | Projectile | Prop | Static | Volume |
|---|---|---|---|---|---|---|
| Character | filter | yes | filter | yes | yes | overlap |
| NPC | | yes | filter | yes | yes | overlap |
| Projectile | | | no | yes | yes | no |
| Prop | | | | yes | yes | overlap |

"filter" = goes through the pair filter (ghost, immunity, friendly fire). "overlap" = detected, no push.

**Pair filter** (ghost and immunity)

- With ≤ 64 players per instance, each player keeps a 64-bit mask "players I don't collide with".
  Check = one bit. Symmetric: set both masks.
- Projectiles carry their owner id; the filter checks the owner's relation to the target.
- Cost (measured): **1.2 ns** with the bitmask vs **13 ns** with a Go map. Expiry ticks live in a 64×64 array.

**Narrowphase** (do they really touch) — measured

| Test | Cost |
|---|---|
| sphere–sphere | 2.5 ns |
| capsule–capsule (segment–segment distance) | 75 ns, 0.03 mm max error vs float64 |
| ray–sphere | 15 ns |
| capsule–box | est. 100–200 ns |

**Missing:** cheap early-outs before expensive tests. Measured: projectiles inside a brawl went from
12 989 capsule tests to 583 with a 3 ns bounding-box check, and the brawl tick from 1.7 ms to 1.2 ms.

### 2.9 Collision response (contact solver)

For each contact, in order:

1. If the bodies are approaching, apply a normal impulse
   `j = −(1 + e)·v_n / (1/m_a + 1/m_b)`, where `v_n` is relative velocity along the contact normal.
2. Friction impulse along the tangent, clamped to `μ·j`.
3. Push the bodies apart by ~80 % of the penetration, ignoring the first 1 cm ("slop") to avoid jitter.

- Repeat over all contacts N times (iterations). More iterations = less jitter in crowds, more cost.
  Measured brawl: 4 iterations 1.7 ms, 8 iterations 2.5 ms.
- **Skip:** gameplay that depends on stable stacks of boxes. Stacking is the hardest thing in rigid-body
  physics and eats iterations.

Cost (measured): **~255 ns per contact per iteration** (170 ns with fast sqrt). In a brawl the solver is
about half of the tick.

### 2.10 Attacks: beams and projectiles

- **Beam = raycast.** Walk the grid cells along the ray in order (3D DDA, "Amanatides & Woo"), test the
  bodies in each cell, stop at the first hit. Visiting cells in order is what implements "hit the first
  entity found", and it lets you stop early.
- **Projectile = moving sphere.** Each tick, sweep the segment from previous to new position against
  nearby capsules. The first hit along the segment wins and the projectile ends.
- **Missing: tunneling.** Anything that moves more than its radius per tick can pass through thin
  things. A 0.4 m capsule tunnels above 12 m/s at 30 Hz. Knockback launches can exceed that: sweep fast
  characters too, or keep the external cap low.

Cost (measured): 75 ns per (projectile, nearby capsule) test after a 3 ns early-out. Brawl: 583 tests ≈ 45 µs.

### 2.11 Impact damage

Proposal:

```
Δv_i     = J / m_i                         change of velocity body i suffers in the impact
damage_i = k × max(0, Δv_i − Δv_safe)²
```

- **Why Δv and not speed:** both bodies receive the same impulse `J` (Newton's third law), but the
  lighter one changes velocity more, so it gets hurt more. A wall has infinite mass: hitting it at
  speed `v` with `e ≈ 0` gives `Δv ≈ v`.
- Only the normal component counts: glancing hits barely hurt.
- **Fall damage is the same formula** against the ground. A fall from h meters lands at `√(2gh)`.
  With `Δv_safe = 8 m/s`: safe up to 3.3 m; a 10 m fall lands at 14 m/s → damage.
- **Wall slam:** knockback into a wall gives a large `J` → damage. Positioning becomes a weapon.
- Resting contact never hurts: gravity adds 0.33 m/s per tick, far below `Δv_safe`.
- Accumulate `J` per contact over the solver iterations; compute damage once per contact per tick.
- Ghost pairs never collide → no damage. Immune pairs collide, but the immune side takes 0.
- **Missing: kill attribution.** If A knocks B into a wall and the wall kills B, who is the killer for
  the ghost rule? Proposal: the last player who damaged or knocked back B in the last 5 s.

Cost (est.): ~5 ns per contact, reusing the solver's impulse. Free.

### 2.12 Character controller

Most of the "feel" of the game lives here.

- Characters are **dynamic bodies** (mass, velocity, impulses) with **rotation locked**. Facing (yaw)
  and aim (pitch) are inputs, not physics.
- Movement: accelerate toward the input's target velocity with a max ground acceleration; reduced
  acceleration in the air (air control).
- **Grounded check:** ground within ~5 cm below, with slope ≤ max walkable (e.g. 45°).
- **Steps:** climb ledges up to ~0.3 m without jumping.
- **Ground snapping:** stick to the ground walking downhill, or characters "fly" off slopes.
- **Jump:** `v0 = √(2gh)`, only when grounded. Optional coyote time (a few ticks of grace after leaving
  a ledge).
- Rejected alternative: kinematic "move and slide" controllers (common in shooters). They make
  mass-based pushing, knockback and impact damage much harder, which are core here.

Cost (est.): ground query against the heightmap 10–20 ns per character; steps and slopes add tests
against static boxes.

### 2.13 Static world

- Terrain as a heightmap; structures as static boxes and capsules with `1/m = 0`.
- A separate static grid built once at load; the dynamic grid is rebuilt every tick.
- Pipeline: Blender collision primitives → export → loader. Render assets never reach the server.

### 2.14 Rotation (deferred on purpose)

- Full rigid bodies need orientation (quaternion), angular velocity, torque, inertia tensor, and
  contacts with angular terms. **Est.** 3–5× more cost per contact and narrowphase, and that is where
  most stability bugs live.
- Characters never need it. Props stay spheres/capsules until a mechanic truly needs tumbling boxes.
- The client can fake rotation visually (a rolling barrel) without the server simulating it.

### 2.15 Volumes and triggers

- Non-solid zones detected by overlap: drag (foliage, mud, water), damage zones, future wind shelters.
- **Buoyancy** (optional): `F = ρ_water · V_submerged · g`; for a capsule, approximate the submerged
  volume from the depth.

### 2.16 Sleeping

- Bodies at rest for N ticks stop being integrated until something touches them. Saves CPU on props;
  characters rarely sleep. Do it in the performance pass.

---

## 3. Excluded

| Item | Why |
|---|---|
| Atmospheric pressure | Your call; no gameplay effect |
| Temperature | Your call |
| Wind turbulence | Your call; future wind = zones + cone shelters |
| Per-branch / per-leaf drag | A volume with a coefficient gives the same gameplay |
| Cloth, hair, particles, debris, ragdolls, destruction fragments | **Cosmetic → client only.** Not authoritative, not deterministic, zero server cost |
| Fluids, soft bodies | Orders of magnitude more cost than the game needs |
| Mesh–mesh collision for moving objects | Use primitives |
| Stable stacking | Solver hell; design around it |
| Real air drag on characters | The external speed cap does the job |
| Real friction for walking | Tuned acceleration feels better |
| Floats in simulation state | Breaks determinism (only exception: the sqrt guess in section 0) |

Rule of thumb: **if it doesn't change who wins, who gets hit or where someone ends up, it runs only on the client.**

---

## 4. Cost model

### Scenario per instance

264 characters (64 players + 200 NPCs), 50 props, 300 projectiles in flight, 10 beams per tick, 30 Hz.

### Full tick (measured)

| Scenario | 4 solver iterations | + projectile early-out | + fast sqrt |
|---|---|---|---|
| Typical: 12 groups spread over 400 × 400 m | 0.15 ms | — | 0.11 ms |
| Brawl: all 264 packed, 200 projectiles inside | 1.70 ms | 1.18 ms | 0.91 ms |
| Brawl, 8 solver iterations | 2.50 ms | — | — |

The budget is 33.3 ms per tick. The worst brawl uses **3–8 % of one core**.
Physics is not the bottleneck at this scale.

What would break the budget: full rotation on many props, mesh colliders, 20+ solver iterations,
projectiles without early-outs (+45 % measured), and N² loops instead of the grid.

### Micro-costs (measured)

| Operation | ns |
|---|---|
| `Mul` Q16 | 3.3 |
| `Div` Q16 | 10 |
| `isqrt` Newton / float guess | 56 / 3 |
| normalize (Newton) | 85 |
| integrate one body | 4 (≈100 when over the speed cap) |
| sphere–sphere | 2.5 |
| capsule–capsule | 75 |
| ray–sphere | 15 |
| contact resolve (normal + friction + correction) | 255 (170 with fast sqrt) |
| ghost check: bitmask / Go map | 1.2 / 13 |
| sin/cos table lookup | 0.8 |

### The cost that will hit first: bandwidth

Not physics, but driven by the same entity counts:

- Naive JSON snapshots: 614 entities × ~50 B ≈ 30 KB, × 30 Hz × 64 players ≈ **59 MB/s** (~470 Mbit/s).
  Impossible on a cheap VPS.
- With interest management (only ~80 relevant entities per player), binary quantized (~12 B/entity)
  and a 20 Hz send rate: ≈ 19 KB/s per player, **≈ 1.2 MB/s per full instance** (~10 Mbit/s),
  ≈ 3 TB/month if full 24/7.
- Cheap plans include ~0.5–1 TB/month (DigitalOcean basic) or 20 TB (Hetzner EU). Bandwidth is a real
  line item. Details belong in `Shared/protocol`.

---

## 5. Open questions

- Kill attribution for environment kills (wall slam, fall after knockback): last attacker within 5 s?
- Do body collisions and slams between **allies** deal impact damage?
- Do ghosts block third-party projectiles? (also in `GameDesign/core-combat.md`)
- Do NPC kills create ghost relations, or only PvP kills?
- The killer's projectiles vs the victim (ghost pair): pass through, or blocked with no damage?
- Max players per instance: ≤ 64 keeps the ghost filter to one `uint64` per player.
- Character mass range: sets how strong knockback differences feel.
