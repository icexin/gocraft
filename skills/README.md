# GoCraft -- Skills Progression

A learning path for building and extending GoCraft, ordered from foundational concepts to advanced features. Each skill builds on previous ones.

## Tier 1: Foundation (understand the engine)

| # | Skill | Key Files | Prerequisite |
|---|-------|-----------|-------------|
| 1.1 | OpenGL rendering pipeline | `render.go`, `block.vert`, `block.frag` | None |
| 1.2 | GLSL shader language | `block.vert`, `block.frag`, `line.*`, `player.*` | 1.1 |
| 1.3 | Vertex buffer objects and mesh building | `cube.go`, `render.go` (Mesh type) | 1.1 |
| 1.4 | Texture atlas and UV mapping | `item.go`, `texture.png`, `build_atlas.py` | 1.3 |
| 1.5 | Camera math (view/projection matrices) | `camera.go`, `render.go` (perspective/lookAt) | 1.1 |
| 1.6 | Chunk data structure and world storage | `chunk.go`, `store.go`, `world.go` | None |
| 1.7 | BoltDB persistence layer | `store.go` | 1.6 |
| 1.8 | Procedural terrain with simplex noise | `world.go` (makeChunkMap), `math.go` | 1.6 |

## Tier 2: Game Mechanics (modify behavior)

| # | Skill | Key Files | Prerequisite |
|---|-------|-----------|-------------|
| 2.1 | Input handling (keyboard + mouse) | `main.go` (callbacks, handleKeyInput) | 1.5 |
| 2.2 | Collision detection (AABB) | `world.go` (CollideStep, CollideStepped) | 1.6, 2.1 |
| 2.3 | Ray casting / hit testing | `world.go` (HitTest) | 1.6, 2.1 |
| 2.4 | Block placement and destruction | `main.go` (onMouseButtonCallback), `world.go` | 2.2, 2.3 |
| 2.5 | Hidden surface removal | `cube.go`, `render.go` (chunk mesh building) | 1.3, 1.6 |
| 2.6 | Frustum culling | `render.go` (frustum extraction, chunk visibility) | 1.5, 1.6 |
| 2.7 | Fog and distance attenuation | `block.vert`, `block.frag` | 1.2 |

## Tier 3: New Features (build the specs)

| # | Skill | Spec | Prerequisite |
|---|-------|------|-------------|
| 3.1 | Day/night cycle (dynamic lighting) | [001](../specs/001-day-night-cycle.md) | 1.2, 2.7 |
| 3.2 | HUD rendering (2D overlay, text) | [002](../specs/002-hud-crosshair-hotbar.md) | 1.1, 1.4 |
| 3.3 | Block selection outline | [003](../specs/003-block-selection-outline.md) | 1.3, 2.3 |
| 3.4 | Sprint and crouch mechanics | [004](../specs/004-sprinting-crouching.md) | 2.1, 2.2 |
| 3.5 | Procedural audio synthesis | [005](../specs/005-procedural-audio.md) | 2.4 |
| 3.6 | Persistent world edits + undo | [006](../specs/006-persistent-world-edits.md) | 1.7, 2.4 |

## Tier 4: Advanced (stretch goals)

| # | Skill | Prerequisite |
|---|-------|-------------|
| 4.1 | Multiplayer networking (RPC protocol) | 1.7, `rpc.go` |
| 4.2 | Shadow mapping | 1.2, 3.1 |
| 4.3 | Water rendering (transparency, refraction) | 1.2, 2.5 |
| 4.4 | Chunk LOD (level of detail for distant terrain) | 2.5, 2.6 |
| 4.5 | Entity/mob system | 2.2, 3.5 |
| 4.6 | Inventory and crafting UI | 3.2 |

## Recommended Build Order

For someone implementing all 6 specs, the optimal order minimizes rework and maximizes testability:

```
1. Block Selection Outline (003) -- small scope, immediate visual payoff
2. Sprint + Crouch (004)         -- pure game logic, no rendering changes beyond FOV
3. Persistent World Edits (006)  -- fixes real bugs (boundary dirtying), adds undo
4. Day/Night Cycle (001)         -- shader work, builds on lighting understanding
5. HUD + Hotbar (002)            -- new rendering subsystem, most complex
6. Procedural Audio (005)        -- independent system, can be done any time
```

## How to Use This

1. Start by reading `specs/current-mechanics.md` to understand the full engine
2. Pick a Tier 1 skill and read the corresponding source files
3. Write a small modification (change a constant, add a log, adjust a threshold) to verify understanding
4. Move to Tier 2 once comfortable with the rendering and data model
5. Implement a Tier 3 spec following the acceptance criteria as a checklist
6. Run `go test ./...` after every change to ensure nothing breaks
