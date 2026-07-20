package main

import "math"

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
	case 1:
		return MaterialDirt
	case 2:
		return MaterialSand
	case 3:
		return MaterialGravel
	case 4:
		return MaterialStone
	case 5:
		return MaterialWood
	case 6:
		return MaterialStone
	case 7:
		return MaterialStone
	case 8:
		return MaterialStone
	case 9:
		return MaterialStone
	case 10:
		return MaterialGlass
	case 11, 12, 13, 14:
		return MaterialStone
	case 15:
		return MaterialDirt
	case 16:
		return MaterialDirt
	default:
		return MaterialStone
	}
}

type SoundGenerator interface {
	Sample(t float64) float32
}

type FootstepSound struct {
	material BlockMaterial
	seed     float64
}

func NewFootstepSound(mat BlockMaterial) *FootstepSound {
	return &FootstepSound{material: mat, seed: 0}
}

func (s *FootstepSound) Sample(t float64) float32 {
	switch s.material {
	case MaterialStone:
		freq := 140.0
		noise := pseudoNoise(t*freq+s.seed) * float32(math.Exp(-t*30))
		return noise * 0.6
	case MaterialDirt:
		freq := 100.0
		attack := float32(math.Min(t*20, 1.0))
		noise := pseudoNoise(t*freq+s.seed) * float32(math.Exp(-t*15)) * attack
		return noise * 0.5
	case MaterialWood:
		freq := 300.0
		resonance := float32(math.Sin(2*math.Pi*freq*t)) * float32(math.Exp(-t*25))
		click := pseudoNoise(t*200+s.seed) * float32(math.Exp(-t*50))
		return (resonance*0.6 + click*0.4) * 0.5
	case MaterialSand:
		freq := 200.0
		noise := pseudoNoise(t*freq+s.seed) * float32(math.Exp(-t*12))
		return noise * 0.4
	case MaterialGlass:
		freq := 1000.0
		sine := float32(math.Sin(2*math.Pi*freq*t)) * float32(math.Exp(-t*40))
		return sine * 0.3
	case MaterialGravel:
		noise := pseudoNoise(t*250+s.seed) * float32(math.Exp(-t*20))
		crackle := pseudoNoise(t*800+s.seed) * float32(math.Exp(-t*30)) * 0.3
		return (noise + crackle) * 0.5
	default:
		return pseudoNoise(t*150+s.seed) * float32(math.Exp(-t*25)) * 0.5
	}
}

type BlockBreakSound struct {
	material BlockMaterial
}

func NewBlockBreakSound(mat BlockMaterial) *BlockBreakSound {
	return &BlockBreakSound{material: mat}
}

func (s *BlockBreakSound) Sample(t float64) float32 {
	burstIdx := int(t / 0.05)
	localT := t - float64(burstIdx)*0.05
	if burstIdx >= 4 || localT > 0.02 {
		return 0
	}
	freq := 200.0 - float64(burstIdx)*30
	noise := pseudoNoise(localT*freq) * float32(math.Exp(-localT*60))
	decay := float32(1.0 - float64(burstIdx)*0.2)
	return noise * decay * 0.5
}

type BlockPlaceSound struct {
	material BlockMaterial
}

func NewBlockPlaceSound(mat BlockMaterial) *BlockPlaceSound {
	return &BlockPlaceSound{material: mat}
}

func (s *BlockPlaceSound) Sample(t float64) float32 {
	freq := 150.0
	switch s.material {
	case MaterialWood:
		freq = 200
	case MaterialGlass:
		freq = 300
	case MaterialSand:
		freq = 100
	}
	sine := float32(math.Sin(2*math.Pi*freq*t)) * float32(math.Exp(-t*15))
	thud := pseudoNoise(t*100) * float32(math.Exp(-t*30)) * 0.3
	return (sine*0.7 + thud) * 0.4
}

type WindSound struct {
	phase float64
}

func (s *WindSound) Sample(t float64) float32 {
	modulation := float32(math.Sin(t*0.1)*0.5 + 0.5)
	noise := pseudoNoise(t*80 + s.phase)
	return noise * modulation * 0.05
}

func pseudoNoise(t float64) float32 {
	x := math.Sin(t*12.9898+78.233) * 43758.5453
	return float32(x - math.Floor(x))*2 - 1
}
