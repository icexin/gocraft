package main

import (
	"math"
	"sync"
)

const (
	blockStone      = 4
	blockWood       = 5
	blockCobble     = 6
	blockStoneBrick = 7
	blockBrick      = 8
	blockSandstone  = 9
	blockGlass      = 10
	blockGrass      = 1
)

type StructureBuilder struct {
	blocks map[Vec3]int
}

func NewStructureBuilder() *StructureBuilder {
	return &StructureBuilder{blocks: make(map[Vec3]int)}
}

func (b *StructureBuilder) Set(x, y, z, tp int) {
	b.blocks[Vec3{x, y, z}] = tp
}

func (b *StructureBuilder) FillBox(x1, y1, z1, x2, y2, z2, tp int) {
	for x := x1; x <= x2; x++ {
		for y := y1; y <= y2; y++ {
			for z := z1; z <= z2; z++ {
				b.Set(x, y, z, tp)
			}
		}
	}
}

func (b *StructureBuilder) FillFloor(y, x1, z1, x2, z2, tp int) {
	for x := x1; x <= x2; x++ {
		for z := z1; z <= z2; z++ {
			b.Set(x, y, z, tp)
		}
	}
}

func (b *StructureBuilder) FillWallX(x, y1, y2, z1, z2, tp int) {
	for y := y1; y <= y2; y++ {
		for z := z1; z <= z2; z++ {
			b.Set(x, y, z, tp)
		}
	}
}

func (b *StructureBuilder) FillWallZ(z, y1, y2, x1, x2, tp int) {
	for y := y1; y <= y2; y++ {
		for x := x1; x <= x2; x++ {
			b.Set(x, y, z, tp)
		}
	}
}

func (b *StructureBuilder) FillCylinder(cx, cz, radius, y1, y2, tp int) {
	r2 := float64(radius) * float64(radius)
	for x := cx - radius; x <= cx+radius; x++ {
		for z := cz - radius; z <= cz+radius; z++ {
			dx := float64(x - cx)
			dz := float64(z - cz)
			if dx*dx+dz*dz <= r2 {
				for y := y1; y <= y2; y++ {
					b.Set(x, y, z, tp)
				}
			}
		}
	}
}

func (b *StructureBuilder) FillHollowCylinder(cx, cz, outerR, innerR, y1, y2, tp int) {
	outer2 := float64(outerR) * float64(outerR)
	inner2 := float64(innerR) * float64(innerR)
	for x := cx - outerR; x <= cx+outerR; x++ {
		for z := cz - outerR; z <= cz+outerR; z++ {
			dx := float64(x - cx)
			dz := float64(z - cz)
			d2 := dx*dx + dz*dz
			if d2 <= outer2 && d2 > inner2 {
				for y := y1; y <= y2; y++ {
					b.Set(x, y, z, tp)
				}
			}
		}
	}
}

func (b *StructureBuilder) FillCone(cx, cz, baseRadius, y1, y2, tp int) {
	height := y2 - y1
	for y := y1; y <= y2; y++ {
		progress := float64(y-y1) / float64(height)
		r := float64(baseRadius) * (1.0 - progress)
		r2 := r * r
		ri := int(r) + 1
		for x := cx - ri; x <= cx+ri; x++ {
			for z := cz - ri; z <= cz+ri; z++ {
				dx := float64(x - cx)
				dz := float64(z - cz)
				if dx*dx+dz*dz <= r2 {
					b.Set(x, y, z, tp)
				}
			}
		}
	}
}

func (b *StructureBuilder) SpiralStaircase(cx, cz, radius, y1, y2, tp int) {
	steps := y2 - y1
	for i := 0; i <= steps; i++ {
		y := y1 + i
		angle := float64(i) * (2 * math.Pi / 8)
		sx := cx + int(math.Round(float64(radius-1)*math.Cos(angle)))
		sz := cz + int(math.Round(float64(radius-1)*math.Sin(angle)))
		b.Set(sx, y, sz, tp)
		sx2 := cx + int(math.Round(float64(radius-2)*math.Cos(angle)))
		sz2 := cz + int(math.Round(float64(radius-2)*math.Sin(angle)))
		if sx2 != sx || sz2 != sz {
			b.Set(sx2, y, sz2, tp)
		}
	}
}

func (b *StructureBuilder) Battlements(y, x1, z1, x2, z2, tp int) {
	for x := x1; x <= x2; x++ {
		if (x-x1)%2 == 0 {
			b.Set(x, y, z1, tp)
			b.Set(x, y, z2, tp)
		}
	}
	for z := z1; z <= z2; z++ {
		if (z-z1)%2 == 0 {
			b.Set(x1, y, z, tp)
			b.Set(x2, y, z, tp)
		}
	}
}

func (b *StructureBuilder) BlockCount() int {
	return len(b.blocks)
}

func (b *StructureBuilder) GetBlock(id Vec3) int {
	if tp, ok := b.blocks[id]; ok {
		return tp
	}
	return 0
}

var (
	castleOnce   sync.Once
	castleBlocks map[Vec3]int
)

const castleBaseY = 12

func getCastleBlocks() map[Vec3]int {
	castleOnce.Do(func() {
		b := NewStructureBuilder()
		BuildCastle(b)
		castleBlocks = make(map[Vec3]int, len(b.blocks))
		for pos, tp := range b.blocks {
			shifted := Vec3{pos.X, pos.Y + castleBaseY, pos.Z}
			castleBlocks[shifted] = tp
		}
	})
	return castleBlocks
}

func applyCastleToChunk(cid Vec3, m map[Vec3]int) {
	cb := getCastleBlocks()
	cx1 := cid.X * ChunkWidth
	cz1 := cid.Z * ChunkWidth
	cx2 := cx1 + ChunkWidth - 1
	cz2 := cz1 + ChunkWidth - 1
	for pos, tp := range cb {
		if pos.X >= cx1 && pos.X <= cx2 && pos.Z >= cz1 && pos.Z <= cz2 {
			if tp == 0 {
				delete(m, pos)
			} else {
				m[pos] = tp
			}
		}
	}
}

func BuildCastle(b *StructureBuilder) {
	const (
		ox, oz         = 0, 0
		keepSize       = 30
		wallHeight     = 8
		towerRadius    = 4
		towerHeight    = 14
		innerTowerR    = 5
		innerTowerH    = 20
		groundY        = 0
		floorY         = 1
	)

	x1, z1 := ox, oz
	x2, z2 := ox+keepSize-1, oz+keepSize-1
	midX := (x1 + x2) / 2
	midZ := (z1 + z2) / 2

	b.FillFloor(groundY, x1-5, z1-5, x2+5, z2+5, blockGrass)
	b.FillFloor(floorY, x1, z1, x2, z2, blockStoneBrick)

	b.FillWallX(x1, floorY+1, floorY+wallHeight, z1, z2, blockStone)
	b.FillWallX(x2, floorY+1, floorY+wallHeight, z1, z2, blockStone)
	b.FillWallZ(z1, floorY+1, floorY+wallHeight, x1, x2, blockStone)
	b.FillWallZ(z2, floorY+1, floorY+wallHeight, x1, x2, blockStone)

	b.Battlements(floorY+wallHeight+1, x1, z1, x2, z2, blockStone)

	wallTop := floorY + wallHeight
	b.FillFloor(wallTop, x1, z1, x1+1, z2, blockStoneBrick)
	b.FillFloor(wallTop, x2-1, z1, x2, z2, blockStoneBrick)
	b.FillFloor(wallTop, x1, z1, x2, z1+1, blockStoneBrick)
	b.FillFloor(wallTop, x1, z2-1, x2, z2, blockStoneBrick)

	gateX1 := midX - 2
	gateX2 := midX + 2
	gateHeight := 4
	for x := gateX1; x <= gateX2; x++ {
		for y := floorY + 1; y <= floorY+gateHeight; y++ {
			b.Set(x, y, z2, 0)
		}
	}
	for x := gateX1; x <= gateX2; x++ {
		b.Set(x, floorY+gateHeight+1, z2, blockSandstone)
	}

	gateTowerL := gateX1 - 1
	gateTowerR := gateX2 + 1
	for y := floorY + 1; y <= floorY+wallHeight+3; y++ {
		b.Set(gateTowerL, y, z2, blockCobble)
		b.Set(gateTowerL, y, z2+1, blockCobble)
		b.Set(gateTowerR, y, z2, blockCobble)
		b.Set(gateTowerR, y, z2+1, blockCobble)
	}

	windowY := floorY + 4
	for z := z1 + 4; z <= z2-4; z += 5 {
		b.Set(x1, windowY, z, blockGlass)
		b.Set(x1, windowY+1, z, blockGlass)
		b.Set(x2, windowY, z, blockGlass)
		b.Set(x2, windowY+1, z, blockGlass)
	}
	for x := x1 + 4; x <= x2-4; x += 5 {
		b.Set(x, windowY, z1, blockGlass)
		b.Set(x, windowY+1, z1, blockGlass)
	}

	corners := [][2]int{
		{x1, z1}, {x2, z1}, {x1, z2}, {x2, z2},
	}
	for _, c := range corners {
		b.FillHollowCylinder(c[0], c[1], towerRadius, towerRadius-2, floorY+1, floorY+towerHeight, blockCobble)
		b.FillFloor(floorY+towerHeight, c[0]-towerRadius, c[1]-towerRadius, c[0]+towerRadius, c[1]+towerRadius, blockStoneBrick)
		b.FillCone(c[0], c[1], towerRadius+1, floorY+towerHeight+1, floorY+towerHeight+5, blockBrick)
	}

	b.FillHollowCylinder(midX, midZ, innerTowerR, innerTowerR-2, floorY+1, floorY+innerTowerH, blockStoneBrick)

	b.FillCylinder(midX, midZ, innerTowerR-2, floorY+1, floorY+1, blockStoneBrick)

	doorX := midX + innerTowerR
	for y := floorY + 1; y <= floorY+3; y++ {
		b.Set(doorX, y, midZ, 0)
		b.Set(doorX-1, y, midZ, 0)
	}

	b.SpiralStaircase(midX, midZ, innerTowerR-1, floorY+2, floorY+innerTowerH-1, blockSandstone)

	midFloorY := floorY + 10
	b.FillCylinder(midX, midZ, innerTowerR-2, midFloorY, midFloorY, blockStoneBrick)
	b.FillCylinder(midX, midZ, innerTowerR-2, floorY+innerTowerH, floorY+innerTowerH, blockStoneBrick)

	b.FillCone(midX, midZ, innerTowerR+1, floorY+innerTowerH+1, floorY+innerTowerH+7, blockBrick)

	topWindowY := floorY + innerTowerH - 2
	for angle := 0.0; angle < 2*math.Pi; angle += math.Pi / 4 {
		wx := midX + int(math.Round(float64(innerTowerR)*math.Cos(angle)))
		wz := midZ + int(math.Round(float64(innerTowerR)*math.Sin(angle)))
		b.Set(wx, topWindowY, wz, blockGlass)
		b.Set(wx, topWindowY+1, wz, blockGlass)
	}

	bridgeY := wallTop
	bridgeZ := midZ
	for x := midX + innerTowerR; x <= x2; x++ {
		b.Set(x, bridgeY, bridgeZ, blockWood)
		b.Set(x, bridgeY, bridgeZ-1, blockWood)
		b.Set(x, bridgeY+1, bridgeZ-2, blockStoneBrick)
		b.Set(x, bridgeY+1, bridgeZ+1, blockStoneBrick)
	}
	for y := bridgeY - 1; y >= floorY+1; y-- {
		for x := midX + innerTowerR; x <= midX+innerTowerR+1; x++ {
			b.Set(x, y, bridgeZ, 0)
			b.Set(x, y, bridgeZ-1, 0)
		}
	}

	b.FillFloor(groundY, x1-3, z2+1, x2+3, z2+8, blockGrass)
	b.FillFloor(floorY, midX-4, z2+1, midX+4, z2+6, blockStoneBrick)
	b.FillWallX(midX-4, floorY+1, floorY+3, z2+1, z2+6, blockStone)
	b.FillWallX(midX+4, floorY+1, floorY+3, z2+1, z2+6, blockStone)
	b.FillWallZ(z2+6, floorY+1, floorY+3, midX-4, midX+4, blockStone)
	for x := midX - 2; x <= midX+2; x++ {
		for y := floorY + 1; y <= floorY+3; y++ {
			b.Set(x, y, z2+6, 0)
		}
	}
}
