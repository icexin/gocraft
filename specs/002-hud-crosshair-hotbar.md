# Spec 002: HUD / Crosshair + Hotbar

## Goal

The player sees a center crosshair, a bottom-center hotbar showing available block types with the selected one highlighted, and an optional F3 debug overlay showing coordinates, FPS, chunk count, and block type under cursor.

## Current State

- Crosshair exists in `LineRender.Draw()` (`render.go:310-330`): two black lines drawn with `Ortho2D` projection, 10px half-size.
- Current block type shown as a 3D rotating preview in bottom-left (`render.go:280-305`), drawn via `BlockRender.drawItem()`.
- Block cycling: E/R keys cycle through `availableItems` (56 items), no visual feedback about which is selected.
- FPS counter exists (`main.go` `FPS` struct) but is only logged, not displayed on screen.
- No text rendering capability exists in the codebase.
- No 2D sprite/quad rendering exists (crosshair uses GL_LINES, item uses 3D mesh).

### Available Block Types (item.go)

56 items in `availableItems`: IDs 1-23 and 32-64. Player cycles through them with E/R.

## Design

### Text Rendering Approach

Rather than adding a full font rendering library, use a **bitmap font atlas** approach:
- Add a 128x64 glyph atlas PNG (`font.png`) containing ASCII 32-127 as 8x8 pixel monospace glyphs (12 columns x 8 rows)
- Render text by drawing textured quads with UV lookups into the glyph atlas
- Reuse the existing `glhf` shader/texture infrastructure

### HUD Shader (new: hud.vert, hud.frag)

New shader pair for 2D textured quads:
```glsl
// hud.vert
in vec2 pos;
in vec2 tex;
uniform mat4 matrix;  // orthographic projection
out vec2 Tex;

// hud.frag
in vec2 Tex;
uniform sampler2D tex;
uniform vec4 tint;  // color tint (white for normal, yellow for highlight)
// discard if alpha < 0.1
```

### HUDRender (new section in render.go or new file hud.go)

```go
type HUDRender struct {
    shader    *glhf.Shader
    fontTex   *glhf.Texture
    blockTex  *glhf.Texture  // reuse existing texture atlas
    width     int
    height    int
}

func NewHUDRender() (*HUDRender, error)
func (h *HUDRender) Draw(state HUDState)
func (h *HUDRender) DrawText(x, y int, text string, scale int)
func (h *HUDRender) DrawHotbar(items []int, selected int)
func (h *HUDRender) DrawCrosshair()
func (h *HUDRender) DrawDebugOverlay(state DebugState)
```

### HUD State

```go
type HUDState struct {
    HotbarItems    []int      // 9 block type IDs in hotbar slots
    SelectedSlot   int        // 0-8
    ShowDebug      bool       // F3 toggle
    FPS            int
    PlayerPos      mgl32.Vec3
    ChunkID        Vec3
    LookingAt      *Vec3      // block under crosshair (nil if none)
    LookingAtType  int        // block type under crosshair
    ChunkCount     int
    FaceCount      int
}
```

### Hotbar Layout

- 9 slots, each 40x40 pixels, 4px gap, centered at bottom of screen
- Each slot draws a dark semi-transparent background quad
- Selected slot has a bright border (2px white outline)
- Inside each slot: the block's top-face texture tile (32x32, from atlas)
- Number keys 1-9 select the slot directly
- E/R still cycle, but now they move the selection cursor

### Crosshair

- Replace current GL_LINES crosshair with a 2D textured crosshair sprite (16x16, rendered from a tile in the font atlas or a dedicated crosshair tile)
- Or keep the line-based crosshair but make it white with black outline for visibility

### Debug Overlay (F3)

Top-left corner, 8px monospace text:
```
FPS: 60
XYZ: -1293.4 / 15.0 / -174.2
Chunk: -41, 0, -6
Looking at: stone (4) @ -1290, 14, -170
Chunks: 144 rendered / 288 cached
Faces: 52,301
```

### Files Changed

| File | Change |
|------|--------|
| `hud.go` (new) | `HUDRender`, text rendering, hotbar, debug overlay |
| `hud.vert` (new) | 2D textured quad vertex shader |
| `hud.frag` (new) | 2D textured quad fragment shader with tint and alpha |
| `font.png` (new) | 8x8 monospace bitmap font atlas |
| `shader.go` | Add `//go:embed` for hud.vert, hud.frag |
| `main.go` | Create `HUDRender`, pass `HUDState`, handle F3 key, handle 1-9 keys, adjust hotbar selection on E/R |
| `render.go` | Remove crosshair from `LineRender.Draw()` (moved to HUD), remove `drawItem()` (moved to HUD) |

### Data Flow

```
Game.Update() -> collect HUDState from camera, world, blockRender
              -> hudRender.Draw(state)
                  -> DrawCrosshair()    (center of screen)
                  -> DrawHotbar()       (bottom center)
                  -> DrawDebugOverlay() (top left, if F3 toggled)
```

## Controls

| Key | Action |
|-----|--------|
| 1-9 | Select hotbar slot directly |
| E | Next slot (wraps) |
| R | Previous slot (wraps) |
| F3 | Toggle debug overlay |

## Acceptance Criteria

1. Crosshair is visible at center of screen at all times
2. Hotbar shows 9 block types as tile icons at bottom center
3. Selected slot has a visible highlight (border or glow)
4. Pressing 1-9 selects the corresponding slot, left/right-click uses that block type
5. F3 toggles a text overlay showing FPS, position, chunk, and target block info
6. `TestHUDStateCollection` -- verifies HUDState fields populate correctly from game state
7. `TestHotbarSlotSelection` -- verifies 1-9 keys and E/R cycling
8. `TestBitmapFontUVMapping` -- verifies UV coordinates for ASCII characters map to correct glyph positions

## Out of Scope (v1)

- Inventory screen (full grid)
- Item tooltips on hover
- Health/hunger bars
- Chat overlay
- Minimap
- Custom hotbar arrangement (drag and drop)
