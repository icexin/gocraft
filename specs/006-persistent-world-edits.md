# Spec 006: Persistent World Edits + Undo

## Goal

Player block edits (place/break) persist reliably across sessions, dirty chunks re-mesh efficiently, and an undo stack lets the player reverse the last 50 actions.

## Current State

- `Store.UpdateBlock()` (`store.go:65`) writes block changes to BoltDB immediately on every edit.
- `World.UpdateBlock()` (`world.go:80`) updates the in-memory chunk sync.Map AND calls `store.UpdateBlock()`, then marks the chunk dirty via `blockRender.DirtyChunk()`.
- `DirtyChunk()` (`render.go:150`) adds the chunk ID to a dirty set, signals `UpdateLoop` goroutine.
- `UpdateLoop()` (`render.go:160-230`) rebuilds meshes for dirty chunks in background, up to 4 per cycle.
- On startup, `World.Chunk()` (`world.go:55-75`) loads from procedural generation, then overlays stored blocks via `store.RangeBlocks()`.
- Camera state persists via `store.UpdatePlayerState()` / `store.GetPlayerState()` on exit/startup.
- `store.db` has `NoSync = true` for performance -- meaning **writes are not fsynced** and can be lost on crash.

### Current Issues

1. **No undo**: Accidental block destruction is permanent.
2. **Crash safety**: `NoSync = true` means recent edits can be lost if the process crashes or is killed.
3. **Adjacent chunk dirtying**: When a block is placed/broken at a chunk boundary, only the owning chunk is marked dirty. Adjacent chunks may show stale face occlusion (invisible faces that should now be visible, or vice versa).
4. **No edit history**: No way to see what the player changed vs. the original world.
5. **Batch writes**: Each block edit triggers a separate BoltDB transaction. High-frequency edits (rapid clicking) create write amplification.

## Design

### Undo Stack (new: history.go)

```go
type EditAction struct {
    Position  Vec3
    OldType   int   // block type before the edit
    NewType   int   // block type after the edit
    Timestamp float64
}

type EditHistory struct {
    mu       sync.Mutex
    stack    []EditAction   // most recent at end
    maxSize  int            // 50
}

func NewEditHistory(maxSize int) *EditHistory
func (h *EditHistory) Push(action EditAction)
func (h *EditHistory) Pop() (EditAction, bool)
func (h *EditHistory) Len() int
func (h *EditHistory) Clear()
```

### Undo Mechanics

- `Ctrl+Z` triggers undo: pops the last action, restores `OldType` at `Position`
- The restore itself is NOT pushed onto the undo stack (prevents undo loops)
- Undo calls `World.UpdateBlock()` with the old type, which persists + dirties chunk
- Stack is capped at 50 entries. Oldest entries silently drop when full.
- Stack resets on world load (no cross-session undo)

### Adjacent Chunk Dirtying (world.go, render.go)

When a block is updated, check if it's on a chunk boundary (x%32==0, x%32==31, z%32==0, z%32==31). If so, also dirty the neighboring chunk(s):

```go
func (w *World) UpdateBlock(id Vec3, tp int) {
    // ... existing logic ...
    cid := id.Chunkid()
    blockRender.DirtyChunk(cid)

    // Dirty adjacent chunks if at boundary
    local := Vec3{
        ((id.X % ChunkWidth) + ChunkWidth) % ChunkWidth,
        id.Y,
        ((id.Z % ChunkWidth) + ChunkWidth) % ChunkWidth,
    }
    if local.X == 0 {
        blockRender.DirtyChunk(Vec3{cid.X - 1, 0, cid.Z})
    }
    if local.X == ChunkWidth-1 {
        blockRender.DirtyChunk(Vec3{cid.X + 1, 0, cid.Z})
    }
    if local.Z == 0 {
        blockRender.DirtyChunk(Vec3{cid.X, 0, cid.Z - 1})
    }
    if local.Z == ChunkWidth-1 {
        blockRender.DirtyChunk(Vec3{cid.X, 0, cid.Z + 1})
    }
}
```

### Write Batching (store.go)

Replace per-edit BoltDB transactions with a batched write queue:

```go
type WriteQueue struct {
    mu      sync.Mutex
    pending map[Vec3]int
    ticker  *time.Ticker  // flush every 500ms
    store   *Store
}

func (q *WriteQueue) Enqueue(id Vec3, tp int)
func (q *WriteQueue) Flush()  // batch-writes all pending in one BoltDB transaction
```

- Edits accumulate for up to 500ms before flushing
- `store.Close()` always flushes remaining writes
- Reduces transaction overhead from N transactions to 1 per 500ms window

### Crash Safety

- Change `NoSync` from `true` to `false` for the `block` bucket writes (accept slight performance hit for durability)
- OR: keep `NoSync = true` but add an explicit `db.Sync()` call after each flush batch
- Recommendation: explicit `db.Sync()` after flush -- best balance of performance and safety

### Files Changed

| File | Change |
|------|--------|
| `history.go` (new) | `EditHistory`, `EditAction`, undo stack logic |
| `main.go` | Capture old block type before edit, push to EditHistory, handle Ctrl+Z for undo |
| `world.go` | Adjacent chunk dirtying in `UpdateBlock()` |
| `store.go` | Add `WriteQueue` for batched persistence, explicit `Sync()` after flush |
| `render.go` | No changes (DirtyChunk already handles being called multiple times) |

### Data Flow

```
Mouse click (break/place)
    |
    v
Capture old block type -> Push EditAction to EditHistory
    |
    v
World.UpdateBlock(pos, newType)
    |
    +-> chunk.sync.Map update (in-memory)
    +-> writeQueue.Enqueue(pos, newType)  (batched persistence)
    +-> blockRender.DirtyChunk(owning chunk)
    +-> blockRender.DirtyChunk(adjacent chunks if boundary)

Ctrl+Z (undo)
    |
    v
EditHistory.Pop() -> EditAction{pos, oldType}
    |
    v
World.UpdateBlock(pos, oldType)  (same flow as above, but NOT pushed to history)
```

## Controls

| Key | Action |
|-----|--------|
| Ctrl+Z | Undo last block edit |

## Acceptance Criteria

1. Block edits persist across game restart (place a block, quit, restart, block is still there)
2. Breaking a block at a chunk boundary correctly updates adjacent chunk mesh (no invisible face artifacts)
3. Ctrl+Z undoes the last block edit, restoring the previous block type
4. Ctrl+Z can be pressed up to 50 times to undo the last 50 edits
5. Undo does not push to the undo stack (no redo, no loops)
6. Rapid block edits (10+ per second) do not cause frame drops (batched writes)
7. Killing the process with SIGKILL loses at most 500ms of edits (vs potentially all edits before)
8. `TestEditHistoryPushPop` -- verifies LIFO ordering and max size cap
9. `TestEditHistoryOverflow` -- verifies oldest entries drop when exceeding maxSize
10. `TestAdjacentChunkDirty` -- verifies placing block at x%32==0 dirties neighbor chunk
11. `TestWriteQueueBatching` -- verifies multiple enqueues result in single DB transaction
12. `TestWriteQueueFlushOnClose` -- verifies pending writes flush when store closes
13. `TestUndoRestoresBlock` -- integration test: break block, undo, verify block restored

## Out of Scope (v1)

- Redo (Ctrl+Y / Ctrl+Shift+Z)
- Edit history serialization (persist undo stack across sessions)
- Multiplayer undo (only local edits are undoable)
- Copy/paste regions
- World backup/snapshot system
- Edit timestamps displayed in UI
