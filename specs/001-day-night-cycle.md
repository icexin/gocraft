# Spec 001: Day/Night Cycle

## Goal

The sky, lighting, and fog smoothly transition between day and night on a continuous 10-minute real-time cycle, giving the world a sense of time and atmosphere.

## Current State

- Sky color is a constant set in `SkyColor()` (`world.go:27`): bright blue `(0.49, 0.75, 0.93)` for procedural, darker `(0.1, 0.1, 0.15)` for imported.
- Directional light is hardcoded in `block.vert:16`: `normalize(vec3(-1, 1, -1))`.
- Fog color = sky color, passed as `sky_color` uniform in `render.go`.
- Ambient = 0.5, diffuse contribution = 0.5, both hardcoded in `block.frag`.
- No concept of game time exists anywhere in the codebase.

## Design

### Game Clock (world.go)

Add a `GameClock` struct:

```go
type GameClock struct {
    cycleLength float64  // seconds per full day (default 600 = 10 min)
    startTime   float64  // GLFW time at game start
}

func (c *GameClock) TimeOfDay() float64  // returns 0.0-1.0 (0=midnight, 0.25=sunrise, 0.5=noon, 0.75=sunset)
func (c *GameClock) SunAngle() float64   // returns radians, sun rotates in XY plane
func (c *GameClock) SunDirection() mgl32.Vec3
func (c *GameClock) SkyColor() mgl32.Vec3
func (c *GameClock) AmbientLevel() float32
```

Time-of-day phases:
- **0.00-0.20**: Night (dark blue sky, ambient 0.15, stars optional)
- **0.20-0.30**: Sunrise (interpolate sky from dark blue to orange to bright blue, ambient 0.15 -> 0.5)
- **0.30-0.70**: Day (bright blue sky, ambient 0.5)
- **0.70-0.80**: Sunset (interpolate sky from bright blue to orange to dark blue, ambient 0.5 -> 0.15)
- **0.80-1.00**: Night (dark blue sky, ambient 0.15)

### Shader Changes (block.vert, block.frag)

Add uniform `float ambient_level` to block shader. Replace hardcoded 0.5 ambient.

Add uniform `vec3 light_dir` to block vertex shader. Replace hardcoded `normalize(vec3(-1,1,-1))`.

Fragment shader:
```glsl
uniform float ambient_level;
// ...
float light = ambient_level + (1.0 - ambient_level) * diff;
```

### Render Integration (render.go)

In `BlockRender.Draw()`:
- Pass `gameClock.SunDirection()` as `light_dir` uniform
- Pass `gameClock.AmbientLevel()` as `ambient_level` uniform
- Pass `gameClock.SkyColor()` as `sky_color` uniform
- Use `gameClock.SkyColor()` for `glClearColor` in `Game.Update()`

### Files Changed

| File | Change |
|------|--------|
| `world.go` | Add `GameClock` struct and methods, replace `SkyColor()` |
| `block.vert` | Add `light_dir` uniform, replace hardcoded light direction |
| `block.frag` | Add `ambient_level` uniform, replace hardcoded 0.5 |
| `render.go` | Pass new uniforms in `BlockRender.Draw()`, add `light_dir` and `ambient_level` to shader uniform spec |
| `main.go` | Create `GameClock` in `NewGame()`, pass to `Game.Update()` clear color |

### Data Flow

```
GLFW time -> GameClock.TimeOfDay() -> SunDirection(), SkyColor(), AmbientLevel()
                                          |              |              |
                                     block.vert     glClearColor    block.frag
                                     (light_dir)    (sky_color)     (ambient)
```

## Controls

None -- automatic. Could add a debug key (F7) to pause/fast-forward time.

## Acceptance Criteria

1. Sky transitions smoothly through blue -> orange -> dark blue -> orange -> blue over 10 minutes
2. Block lighting dims at night (ambient drops from 0.5 to 0.15)
3. Sun direction rotates, casting shadows from different angles throughout the day
4. Fog color matches sky color at all times
5. `TestGameClockTimeOfDay` -- verifies TimeOfDay wraps correctly at cycle boundaries
6. `TestGameClockSkyColorInterpolation` -- verifies sky colors at noon, midnight, sunrise, sunset
7. `TestGameClockSunDirection` -- verifies sun at noon points roughly downward, at midnight roughly upward
8. Imported worlds also get the day/night cycle (override static dark sky)

## Out of Scope (v1)

- Stars at night
- Moon rendering
- Torch/point light sources
- Shadow mapping
- Block light levels (Minecraft-style light propagation)
- Sleeping to skip night
