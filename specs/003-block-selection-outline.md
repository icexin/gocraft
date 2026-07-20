# Spec 003: Block Selection Outline

## Goal

A wireframe cube outline highlights the block the player is currently looking at, giving clear visual feedback for which block will be broken or where a new block will be placed.

## Current State

- `LineRender.Draw()` (`render.go:310-360`) already draws a wireframe around the targeted block.
- `makeWireFrameData()` (`cube.go:150-200`) generates line vertices for visible faces at 1.06x scale.
- The wireframe is drawn as solid black GL_LINES using `line.vert`/`line.frag`.
- `HitTest()` (`world.go:115-130`) ray marches at 0.125 step up to 8 blocks, returns `*Vec3` target and `*Vec3` previous position.
- The wireframe updates every frame based on `game.CurrentBlockid()` which calls `HitTest`.

### Current Issues

1. The wireframe is always black -- invisible against dark blocks (obsidian, dark stone, coal ore)
2. Scale factor 1.06 causes z-fighting artifacts at certain angles
3. No visual indicator for the placement position (where right-click would place a block)
4. Lines are 1px thin, hard to see at distance

## Design

### Improved Wireframe Shader (line.vert, line.frag)

Replace solid black with a **pulsing white-black outline** for visibility against any background:

```glsl
// line.frag (updated)
uniform float time;
out vec4 color;
void main() {
    float pulse = sin(time * 4.0) * 0.5 + 0.5;  // 0-1 oscillation
    float brightness = mix(0.1, 0.9, pulse);
    color = vec4(brightness, brightness, brightness, 1.0);
}
```

Add `time` uniform to line shader, passed from `glfw.GetTime()`.

### Wireframe Geometry Improvements

- Use offset of 0.002 (not 1.06x scale) -- render at exact block bounds + tiny epsilon to prevent z-fighting
- Draw all 12 edges of the cube (not just visible faces) for a complete box outline
- Add a second, slightly larger wireframe in contrasting color for outline effect (optional, may be overkill)

### Placement Preview

When `HitTest` returns a valid target + previous position:
- Draw a second wireframe at the **previous** position (where right-click would place) in a distinct color (green-tinted)
- Only show placement preview when right-click would succeed (i.e., previous position is air)

### Line Width

- Use `gl.LineWidth(2.0)` for the selection wireframe (wider than the default 1.0)
- Note: line width >1.0 is deprecated in OpenGL core profile on some drivers. If not supported, fall back to drawing thin quads (two triangles per line segment) -- but try `LineWidth(2)` first since most desktop drivers still support it.

### Files Changed

| File | Change |
|------|--------|
| `line.frag` | Add `time` uniform, pulsing brightness |
| `render.go` | Pass `time` uniform, draw placement preview wireframe, adjust wireframe offset, set line width |
| `cube.go` | Add `makeFullWireFrameData()` that draws all 12 edges regardless of adjacent blocks |
| `main.go` | Pass previous hit position to `LineRender` for placement preview |

### Data Flow

```
HitTest(pos, dir) -> (targetBlock, prevBlock)
                         |              |
                    wireframe       placement preview
                    (pulsing)       (green, if air)
```

## Controls

None -- purely visual feedback based on where the player is looking.

## Acceptance Criteria

1. Wireframe is visible against both light and dark blocks (pulsing contrast)
2. No z-fighting -- wireframe lines don't flicker when viewed head-on
3. Placement preview wireframe appears at the position where right-click would place a block
4. Wireframe disappears when not looking at any block (beyond 8-block range)
5. `TestWireframeDataEdgeCount` -- verifies `makeFullWireFrameData` produces exactly 12 edges * 2 vertices = 24 vertices
6. `TestWireframeOffsetNoZFighting` -- verifies offset epsilon is applied correctly
7. Performance: wireframe rendering adds <0.1ms per frame (no mesh rebuild, just uniform updates)

## Out of Scope (v1)

- Block breaking animation (crack overlay texture)
- Block breaking progress bar
- Highlight face only (instead of full cube wireframe)
- Transparent block placement ghost preview (semi-transparent textured cube)
