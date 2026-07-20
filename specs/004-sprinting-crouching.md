# Spec 004: Sprinting + Crouching

## Goal

The player can sprint (faster movement, narrower FOV effect) and crouch (slower movement, prevents falling off edges), adding depth to navigation and building.

## Current State

- Walking speed: `delta` passed directly to `Camera.OnMoveChange()` (`camera.go:90-115`), effective speed ~4.3 blocks/sec.
- Flying speed: 5x walking speed (`camera.go:96`).
- Jump: `vy = 8` when grounded (`main.go` `handleKeyInput`).
- Gravity: `vy -= dt * 20`.
- No sprint or crouch state exists.
- No FOV manipulation exists -- projection matrix uses fixed `mgl32.Perspective(radian(65), ...)` in `render.go:260`.
- Collision uses `CollideStepped` with player hitbox padding of 0.25 on each side.
- No edge detection or edge-cling mechanic.

## Design

### Movement State Machine (camera.go)

Add to `Camera`:

```go
type MovementMode int
const (
    ModeWalk MovementMode = iota
    ModeSprint
    ModeCrouch
)

// New Camera fields:
movementMode   MovementMode
sprintFOV      float32   // target FOV when sprinting (75 vs normal 65)
currentFOV     float32   // smoothly interpolated
crouchHeight   float32   // Y offset when crouching (-0.3)
```

Speed multipliers:
| Mode | Walk Speed | Fly Speed |
|------|-----------|-----------|
| Walk | 1.0x | 5.0x |
| Sprint | 1.6x | 8.0x |
| Crouch | 0.5x | 2.5x |

### Sprint Mechanics

- **Activation**: Hold Left Ctrl while moving forward (W key must be held)
- **Deactivation**: Release Ctrl, release W, or collide with a wall
- **FOV effect**: FOV smoothly lerps from 65 to 75 degrees over 0.2 seconds when sprinting, back to 65 when stopping
- **Hunger**: No hunger system exists, so sprinting is unlimited (for now)
- **Jump-sprint**: Jumping while sprinting maintains sprint speed

### Crouch Mechanics

- **Activation**: Hold Left Shift
- **Deactivation**: Release Shift
- **Edge prevention**: When crouching, collision detection is extended to check the block below the player's feet at the next horizontal position. If that block is air, the horizontal movement is clamped to the edge.
- **Camera offset**: Camera Y position lowers by 0.3 units when crouching (smooth transition over 0.1 seconds)
- **Hitbox**: No hitbox height change in v1 (player remains same collision size)

### Edge Prevention Algorithm (world.go)

```go
func CollideSteppedCrouch(bg BlockGetter, from, to mgl32.Vec3) (mgl32.Vec3, bool) {
    result, grounded := CollideStepped(bg, from, to)
    if !grounded {
        return result, grounded
    }
    // Check if the block below the destination feet is air
    feetY := int(math.Floor(float64(result.Y()))) - 1
    for _, offset := range edgeCheckOffsets {
        checkPos := Vec3{
            int(math.Floor(float64(result.X()) + offset[0])),
            feetY,
            int(math.Floor(float64(result.Z()) + offset[1])),
        }
        if bg.GetBlock(checkPos) == 0 {
            // Clamp: don't allow movement past the edge
            result = from  // revert horizontal, keep vertical
            break
        }
    }
    return result, true
}
```

### FOV Interpolation (render.go)

In `BlockRender.Draw()`, replace fixed `radian(65)` with `radian(camera.CurrentFOV())`.

FOV lerp in `Camera.Update(dt)`:
```go
func (c *Camera) Update(dt float32) {
    targetFOV := float32(65.0)
    if c.movementMode == ModeSprint {
        targetFOV = 75.0
    }
    c.currentFOV = mix(c.currentFOV, targetFOV, min(1.0, dt*10.0))
}
```

### Files Changed

| File | Change |
|------|--------|
| `camera.go` | Add `MovementMode`, `currentFOV`, `crouchHeight`, `Update(dt)`, `CurrentFOV()`, speed multiplier logic |
| `main.go` | Handle Shift/Ctrl key state, call `camera.Update(dt)`, pass movement mode to collision, apply crouch height offset |
| `world.go` | Add `CollideSteppedCrouch()` with edge-prevention logic |
| `render.go` | Use `camera.CurrentFOV()` instead of hardcoded 65 in perspective matrix |

### Data Flow

```
Key state (Shift/Ctrl) -> Camera.movementMode
                              |
                    +---------+---------+
                    |         |         |
              speed mult   FOV lerp  crouch offset
                    |         |         |
              OnMoveChange  Perspective  Camera Y
                    |         |         |
              CollideStepped  Draw()    Pos()
              or CrouchCollide
```

## Controls

| Key | Action |
|-----|--------|
| Left Shift (hold) | Crouch -- slow movement, edge prevention |
| Left Ctrl (hold) | Sprint -- fast movement, FOV widening (requires W) |

## Acceptance Criteria

1. Holding Ctrl while pressing W increases movement speed by 1.6x
2. Releasing Ctrl or W returns to normal speed
3. FOV smoothly widens to 75 when sprinting, returns to 65 when stopping
4. Holding Shift slows movement to 0.5x and prevents walking off edges
5. Crouching lowers camera by 0.3 units smoothly
6. Crouch edge-prevention works: player stops at block edges when crouching over a ledge
7. Sprint and crouch are mutually exclusive (sprint takes priority if both held)
8. Flying mode: sprint = 8x, crouch = 2.5x (vs normal 5x)
9. `TestSprintSpeedMultiplier` -- verifies 1.6x distance traveled per tick
10. `TestCrouchSpeedMultiplier` -- verifies 0.5x distance traveled per tick
11. `TestCrouchEdgePrevention` -- verifies player cannot walk off a platform edge while crouching
12. `TestFOVInterpolation` -- verifies FOV lerps toward target over multiple ticks
13. `TestSprintRequiresForward` -- verifies sprint deactivates without W key

## Out of Scope (v1)

- Double-tap W to sprint (only Ctrl+W for v1)
- Crouch hitbox height reduction (for fitting under 1-block gaps)
- Sprint particles (puff effects at feet)
- Exhaustion/hunger system limiting sprint duration
- Crouch toggle mode (hold-only for v1)
- Sliding on ice while sprinting
