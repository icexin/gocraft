# Spec 005: Procedural Audio

## Goal

The world has ambient sound and responsive audio feedback: footsteps that vary by surface material, block break/place sounds, and environmental wind, all generated procedurally without external audio asset files.

## Current State

- Zero audio capability in the codebase.
- No audio library in `go.mod`.
- GLFW does not provide audio APIs.
- Block type is known at foot position (via `world.GetBlock`).
- Block break/place events are in `onMouseButtonCallback` (`main.go`).
- Walking state is detectable from `handleKeyInput` movement.

## Design

### Audio Library

Add `github.com/hajimehoshi/oto/v2` -- a pure Go, cross-platform audio output library. It provides a low-level PCM audio writer with no CGo dependencies on Linux (ALSA) and works on macOS/Windows too.

Alternative: `github.com/faiface/beep` -- higher-level, built on oto, with mixers and streaming. Slightly heavier but easier to use for mixing multiple sounds.

Recommendation: **Use `oto/v2` directly** for minimal dependencies and full control over synthesis.

### Audio Engine (new: audio.go)

```go
type AudioEngine struct {
    ctx      *oto.Context
    player   *oto.Player
    mixer    *Mixer
    volume   float32  // 0.0 - 1.0
}

type Mixer struct {
    mu       sync.Mutex
    voices   []*Voice
}

type Voice struct {
    generator  SoundGenerator
    startTime  float64
    duration   float64
    volume     float32
    done       bool
}

type SoundGenerator interface {
    Sample(t float64) float32  // returns -1.0 to 1.0
}

func NewAudioEngine() (*AudioEngine, error)
func (a *AudioEngine) Play(gen SoundGenerator, duration float64, volume float32)
func (a *AudioEngine) Close()
```

### Sound Generators (new: sounds.go)

All sounds are synthesized from basic waveforms -- no audio files needed.

```go
type FootstepSound struct {
    material  BlockMaterial  // stone, dirt, wood, sand, glass
}

type BlockBreakSound struct {
    material  BlockMaterial
}

type BlockPlaceSound struct {
    material  BlockMaterial
}

type WindSound struct {
    // Filtered noise, slowly modulated
}
```

#### Footstep Synthesis by Material

| Material | Waveform | Description |
|----------|----------|-------------|
| Stone | Short noise burst (80-200Hz bandpass), 80ms | Hard tap |
| Dirt/Grass | Low noise burst (60-150Hz), 100ms, soft attack | Soft thud |
| Wood | Filtered click (200-400Hz resonant), 60ms | Hollow knock |
| Sand | Pink noise burst (100-300Hz), 120ms, slow decay | Crunchy shuffle |
| Glass | High sine ping (800-1200Hz), 50ms, fast decay | Light tap |
| Gravel | Noise burst (100-400Hz) + crackle, 100ms | Gravelly crunch |

#### Block Break Synthesis

- 3-4 rapid noise bursts (20ms each) with descending pitch
- Material determines base frequency range
- Total duration: ~200ms

#### Block Place Synthesis

- Single thud: sine wave (100-200Hz) with fast attack, medium decay
- Material determines resonance
- Total duration: ~150ms

#### Wind Ambient

- Continuous pink noise, amplitude modulated by `sin(time * 0.1) * 0.5 + 0.5`
- Volume: 0.05 (very quiet background)
- Pitch shifts slowly with simplex noise for organic feel

### Material Classification (world.go or audio.go)

```go
type BlockMaterial int
const (
    MaterialStone BlockMaterial = iota
    MaterialDirt
    MaterialWood
    MaterialSand
    MaterialGlass
    MaterialGravel
)

func BlockMaterialFromType(blockType int) BlockMaterial {
    switch blockType {
    case 1:  return MaterialDirt    // grass
    case 2:  return MaterialSand    // sand
    case 3:  return MaterialGravel  // gravel
    case 4:  return MaterialStone   // stone
    case 5:  return MaterialWood    // wood
    case 6:  return MaterialStone   // cobblestone
    case 7:  return MaterialStone   // stone brick
    case 8:  return MaterialStone   // brick
    case 9:  return MaterialStone   // sandstone
    case 10: return MaterialGlass   // glass
    // ...
    default: return MaterialStone
    }
}
```

### Footstep Trigger Logic (main.go)

```go
var lastFootstepPos mgl32.Vec3
var footstepAccum  float32

// In handleKeyInput:
if moving && onGround {
    footstepAccum += delta * speed
    if footstepAccum > 2.0 {  // every 2 blocks traveled
        feetBlock := world.Block(NearBlock(pos.Sub(mgl32.Vec3{0, 1, 0})))
        mat := BlockMaterialFromType(feetBlock)
        audio.Play(NewFootstepSound(mat), 0.08, 0.3)
        footstepAccum = 0
    }
}
```

### Integration Points

| Event | Source | Sound |
|-------|--------|-------|
| Walking | `handleKeyInput` (movement detected + grounded) | Footstep (material from block below feet) |
| Block break | `onMouseButtonCallback` left-click | BlockBreak (material from broken block) |
| Block place | `onMouseButtonCallback` right-click | BlockPlace (material from placed block) |
| Ambient | `AudioEngine` background goroutine | Wind (continuous, quiet) |

### Files Changed

| File | Change |
|------|--------|
| `audio.go` (new) | AudioEngine, Mixer, Voice, sample mixing, oto integration |
| `sounds.go` (new) | SoundGenerator implementations (footstep, break, place, wind) |
| `main.go` | Create AudioEngine, trigger footsteps in handleKeyInput, trigger break/place in mouse callback |
| `world.go` | Add `BlockMaterialFromType()` |
| `go.mod` | Add `github.com/hajimehoshi/oto/v2` |

### Data Flow

```
handleKeyInput -> movement + grounded? -> footstepAccum -> Play(FootstepSound)
mouseCallback  -> left click           -> broken block  -> Play(BlockBreakSound)
mouseCallback  -> right click          -> placed block  -> Play(BlockPlaceSound)
background     -> continuous           -> Wind          -> Play(WindSound, loop)
                                                              |
                                                         Mixer.Read()
                                                              |
                                                         oto.Player (PCM out)
```

## Controls

| Key | Action |
|-----|--------|
| (none) | Sounds play automatically based on actions |
| (future: M) | Mute/unmute toggle |

## Acceptance Criteria

1. Footsteps play at regular intervals while walking on ground
2. Different materials produce noticeably different footstep sounds
3. Block break produces a crumbling/impact sound
4. Block place produces a thud sound
5. Quiet wind ambient plays continuously in the background
6. No audio when standing still
7. No audio while flying (no footsteps -- blocks are not underfoot)
8. `TestBlockMaterialMapping` -- verifies all 16 block types map to a valid material
9. `TestFootstepSoundWaveform` -- verifies FootstepSound.Sample() produces non-zero output
10. `TestMixerMultipleVoices` -- verifies mixer sums multiple concurrent sounds correctly
11. `TestSoundDuration` -- verifies voices auto-expire after their duration
12. Audio does not cause frame drops (mixer runs on separate goroutine)

## Out of Scope (v1)

- 3D positional audio (stereo panning based on sound source direction)
- Reverb/echo in caves
- Music / background tracks
- Volume slider UI
- Water splash sounds
- Falling damage sound
- Entity/mob sounds
- Adjustable audio sample rate (fixed at 44100Hz)
