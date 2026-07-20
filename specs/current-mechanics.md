# GoCraft -- Current Game Mechanics

## Overview

GoCraft is a voxel game engine written in Go using OpenGL 3.3 Core Profile. It supports procedural world generation via simplex noise, Minecraft world importing (Anvil format), multiplayer via RPC, first-person camera controls, block placement/destruction, and persistent storage with BoltDB.

---

## Architecture

```
main.go          Entry point, game loop (60Hz tick), input dispatch, window init
world.go         Chunk loading/caching (LRU), collision, hit testing, terrain generation
render.go        Block/line/player rendering, mesh cache, frustum culling
camera.go        First-person Euler-angle camera, flying mode, state serialization
chunk.go         Vec3, Chunk data structure (sync.Map of blocks)
cube.go          Vertex data generation for cubes, plants, wireframes
item.go          Texture atlas UV mapping, block type registry (65 types)
math.go          Simplex noise (2D/3D fBm), trig helpers
shader.go        go:embed for 6 GLSL shader files
store.go         BoltDB persistence (blocks, chunks, camera state)
rpc.go           Multiplayer RPC client (block updates, player sync)
player.go        Remote player rendering with state interpolation
```

### Shaders (3 pairs, embedded via go:embed)

| Pair | Vertex | Fragment | Purpose |
|------|--------|----------|---------|
| Block | `block.vert` | `block.frag` | Textured blocks with directional light, fog, sky-color blending. Magenta = transparent. White pixels get inverted diffuse (cloud/snow glow). |
| Line | `line.vert` | `line.frag` | Solid black lines. Crosshair overlay and block selection wireframe. |
| Player | `player.vert` | `player.frag` | Textured cubes for remote players. No lighting/fog. Magenta = transparent. |

### Dependencies

| Package | Purpose |
|---------|---------|
| `boltdb/bolt` | Embedded key-value store |
| `faiface/glhf` | OpenGL shader/texture helpers |
| `go-gl/gl` | OpenGL 3.3 core bindings |
| `go-gl/glfw` | Window + input (GLFW) |
| `go-gl/mathgl` | Matrix/vector math |
| `hashicorp/golang-lru` | Chunk cache |
| `ojrac/opensimplex-go` | Terrain noise |
| `icexin/gocraft-server` | Multiplayer protocol |

---

## Rendering Pipeline

### Frame Update (`Game.Update()` in main.go)

1. Clear color buffer to sky color, clear depth buffer
2. `blockRender.Draw()` -- bind block shader + texture atlas, iterate visible chunks, draw meshes, draw item preview
3. `lineRender.Draw()` -- draw crosshair (2D ortho), draw wireframe around targeted block (3D perspective)
4. `playerRender.Draw()` -- draw all remote players with interpolated positions
5. Swap buffers, poll GLFW events

### Chunk Mesh Building (`BlockRender.UpdateLoop()` in render.go)

- Background goroutine, triggered by channel signal
- Determines visible chunks via frustum culling (6 extracted clip planes from MVP matrix)
- Sorts: visible chunks first, then by distance to camera
- Rebuilds up to 4 dirty chunk meshes per cycle
- Hidden surface removal: only generates face vertices when adjacent block is air/transparent/plant
- Each face = 6 vertices * 8 floats (pos:3 + tex:2 + normal:3) = 48 floats

### Frustum Culling

- Extracts 6 planes from combined projection * view matrix
- Tests 8 corners of each chunk's AABB (32 wide x 256 tall x 32 deep)
- A chunk is visible if any corner is inside all 6 planes

### Fog

- Vertex shader computes `fog_factor = pow(clamp(dist/fogdis, 0, 1), 4)`
- Fragment shader mixes final color with `sky_color` by `fog_factor`
- `fogdis` uniform = `renderRadius * ChunkWidth` (default 6 * 32 = 192)

### Lighting

- Single directional light: `normalize(vec3(-1, 1, -1))`
- Diffuse = `max(dot(normal, light_dir), 0)`
- Ambient = 0.5, diffuse contribution = 0.5
- Special case: white (1,1,1) pixels use `1 - diff*0.2` for subtle shading (clouds/snow)

### Texture Atlas

- 256x256 PNG, organized as 16x16 grid of 16x16-pixel tiles
- Tile index -> UV: `col = idx % 16`, `row = idx / 16`, half-pixel inset for filtering
- Magenta (255,0,255) = transparent, discarded in fragment shader
- `itemDesc` maps 65 block types to 6 face tile indices [left, right, top, bottom, front, back]
- CC BY-SA 3.0 Minetest Game textures for natural blocks

---

## World System

### Procedural Generation (world.go `makeChunkMap`)

- 2D simplex noise (4 octaves, persistence 0.5, lacunarity 2.0) for terrain height
- Second noise controls max height: `g*32+16`
- Sand floor at y=12
- Plants (flowers, grass) placed with noise thresholds >0.6, >0.7
- Trees: trunk height 7, leaf sphere radius ~3.3, noise threshold >0.79
- Cloud layer at y=64-72 using 3D noise threshold >0.69 (block type 16)

### Imported Worlds (cmd/mcimport)

- Parses Minecraft Java Edition Anvil format (.mca region files)
- Full NBT parser (13 tag types)
- Maps ~310 Minecraft block types to 16 GoCraft block types + 23 plant types
- Fallback heuristic mapping (name contains "stone" -> stone_brick, "wood" -> wood, etc.)
- Reads level.dat for spawn position
- Scans overworld + nether + end dimensions
- Outputs to BoltDB with same format as procedural world

### Chunk Storage

- `Chunk` = sync.Map of `Vec3 -> int` (block position -> block type)
- `ChunkWidth = 32` (32x32 horizontal, unlimited Y)
- `Vec3.Chunkid()` maps block position to chunk ID (Y always 0)
- LRU cache sized to `renderRadius^2 * 4` chunks

### Block Types

| ID | Block | Texture Tiles |
|----|-------|--------------|
| 1 | Grass/Dirt | side=16, top=32, bottom=0 |
| 2 | Sand | all=1 |
| 3 | Gravel | all=2 |
| 4 | Stone | all=3 |
| 5 | Wood (log/planks) | side=20, top=36, bottom=4 |
| 6 | Cobblestone | all=5 |
| 7 | Stone Brick | all=6 |
| 8 | Brick | all=7 |
| 9 | Sandstone | side=24, top=40, bottom=8 |
| 10 | Glass | all=9 |
| 11 | Diamond | all=10 |
| 12 | Gold | all=11 |
| 13 | Iron | all=12 |
| 14 | Redstone/Lava | all=13 |
| 15 | Leaves | all=14 |
| 16 | Cloud/Snow | all=15 |
| 17-23 | Plants | Cross-billboard sprites (tiles 48-54) |
| 32-64 | Colored blocks | Single-color tiles |

### Block Classification

- `IsPlant(tp)`: types 17-31 (rendered as X-shaped billboards, not cubes)
- `IsTransparent(tp)`: plants, air (0), glass (10), leaves (15), unknown (-1)
- `IsObstacle(tp)`: everything except plants and air; unknown (-1) IS an obstacle

---

## Player Controls

### Movement (main.go `handleKeyInput`)

| Key | Action |
|-----|--------|
| W | Move forward |
| S | Move backward |
| A | Move left (strafe) |
| D | Move right (strafe) |
| Space | Jump (velocity = 8 upward) |
| Tab | Toggle flying mode |
| E | Next block type |
| R | Previous block type |
| Escape | Release mouse cursor |

### Mouse

| Button | Action |
|--------|--------|
| Left click | Destroy targeted block (within 8 blocks, ray march at 0.125 step) |
| Right click | Place current block at previous ray-march position |
| Mouse move | Look around (Euler angles, pitch clamped to +/-89 degrees) |

### Flying Mode

- Toggle with Tab key
- 5x movement speed multiplier
- No gravity applied
- Movement follows full 3D camera front vector (vs walking uses horizontal-only `wfront`)

### Camera (camera.go)

- Euler angle rotation (yaw = `rotatex`, pitch = `rotatey`)
- Sensitivity: 0.14 (mouse delta multiplier)
- Large mouse jumps (>200 pixels) are ignored (prevents cursor warp glitches)
- Default facing: -Z direction (rotatex = -90)
- State serializable to `PlayerState{X, Y, Z, Rx, Ry}`

---

## Physics

### Gravity (main.go `handleKeyInput`)

- `vy -= dt * 20` (gravitational acceleration, 20 units/sec^2)
- Applied only when not flying
- Jump sets `vy = 8` (only when grounded: `onGround == true`)
- Grounded detection via `CollideStepped` return value

### Collision Detection (world.go)

- AABB player hitbox: 0.25 unit padding on all sides from block center
- `CollideStep`: single-step collision, checks 6 adjacent blocks, adjusts position component-wise
- `CollideStepped`: multi-step collision for tunneling prevention, max step size 0.2 units
- Plants are passthrough (not obstacles)
- Unknown blocks (-1, unloaded chunks) ARE obstacles (prevents falling through unloaded terrain)

### Hit Testing (world.go `HitTest`)

- Ray march from camera position along view direction
- Step size: 0.125 units
- Max range: 8 blocks
- Returns: targeted block position + previous step position (for placement)

---

## Persistence (store.go)

### BoltDB Schema

| Bucket | Key Format | Value Format |
|--------|-----------|-------------|
| `block` | `[chunkX:i32, chunkZ:i32, blockX:i32, blockY:i32, blockZ:i32]` (20 bytes) | `[blockType:u32]` (4 bytes) |
| `chunk` | chunk ID bytes | version string |
| `camera` | `"camera"` | `PlayerState{X,Y,Z,Rx,Ry}` binary |

- `NoSync = true` for write performance
- Default DB path: `gocraft.db` (flag `--db`)
- Camera state saved on exit, restored on startup

---

## Multiplayer (rpc.go)

- Optional, enabled with `-s <host:port>` flag (default port 8421)
- Uses `icexin/gocraft-server` RPC protocol over TCP with yamux multiplexing
- `ClientUpdateBlock(id, type)`: broadcasts block changes to all players
- `ClientUpdatePlayerState(state)`: sends position/rotation at 10Hz, receives other player states
- `ClientFetchChunk(id, callback)`: fetches chunk data from server (overrides local generation)
- Remote players rendered as single textured cubes with linear state interpolation

---

## Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-t` | `texture.png` | Texture atlas path |
| `-r` | `6` | Render radius in chunks |
| `-s` | (none) | Multiplayer server address |
| `--db` | `gocraft.db` | Database file path |
| `--imported` | `false` | Use imported world mode (darker sky, skip terrain gen) |
| `--pprof` | (none) | pprof HTTP server port |

---

## Test Coverage

| Package | Tests | Notes |
|---------|-------|-------|
| Main (root) | 36 | Collision, camera, HitTest, Vec3, chunk, texture atlas validation |
| cmd/mcimport | 19 | NBT parsing, block mapping, DB encoding, level.dat, integration |
| cmd/validate | 6 | World validation, block ranges, bounds, spawn |
| **Total** | **61** | All passing on branch `minecraft-import-and-improvements` |

---

## Tools

| Tool | Location | Purpose |
|------|----------|---------|
| mcimport | `cmd/mcimport/` | Import Minecraft Java worlds to GoCraft DB |
| mcparse | `cmd/mcparse/` | Inspect Minecraft region files, scan block types |
| validate | `cmd/validate/` | Validate GoCraft DB (block ranges, collision, spawn) |
| build_atlas.py | root | Assemble texture atlas from individual PNGs |
| analyze_blocks.py | root | Analyze block mapping coverage for imported worlds |
