package main

import (
	"testing"

	"github.com/go-gl/mathgl/mgl32"
)

func (b *StructureBuilder) ApplyTo(bg *MockBlockGetter) {
	for pos, tp := range b.blocks {
		bg.SetBlock(pos, tp)
	}
}

func buildCastleWorld() *MockBlockGetter {
	b := NewStructureBuilder()
	BuildCastle(b)
	bg := NewMockBlockGetter()
	b.ApplyTo(bg)
	return bg
}

func TestCastleBlockCount(t *testing.T) {
	b := NewStructureBuilder()
	BuildCastle(b)
	count := b.BlockCount()
	if count < 3000 {
		t.Errorf("castle block count = %d, expected at least 3000", count)
	}
	if count > 30000 {
		t.Errorf("castle block count = %d, expected less than 30000", count)
	}
	t.Logf("castle total blocks: %d", count)
}

func TestCastleFloorExists(t *testing.T) {
	b := NewStructureBuilder()
	BuildCastle(b)
	for x := 0; x <= 29; x++ {
		for z := 0; z <= 29; z++ {
			tp := b.GetBlock(Vec3{x, 1, z})
			if tp == 0 {
				t.Errorf("missing floor block at (%d, 1, %d)", x, z)
				return
			}
		}
	}
}

func TestCastleWallsExist(t *testing.T) {
	b := NewStructureBuilder()
	BuildCastle(b)
	for y := 2; y <= 9; y++ {
		if b.GetBlock(Vec3{0, y, 15}) == 0 {
			t.Errorf("west wall missing at (0, %d, 15)", y)
		}
		if b.GetBlock(Vec3{29, y, 15}) == 0 {
			t.Errorf("east wall missing at (29, %d, 15)", y)
		}
		if b.GetBlock(Vec3{15, y, 0}) == 0 {
			t.Errorf("north wall missing at (15, %d, 0)", y)
		}
	}
}

func TestCastleGatehouseOpening(t *testing.T) {
	b := NewStructureBuilder()
	BuildCastle(b)
	for x := 12; x <= 16; x++ {
		for y := 2; y <= 5; y++ {
			tp := b.GetBlock(Vec3{x, y, 29})
			if tp != 0 {
				t.Errorf("gatehouse should be clear at (%d, %d, 29), got block type %d", x, y, tp)
			}
		}
	}
}

func TestCastleCentralTowerDoor(t *testing.T) {
	b := NewStructureBuilder()
	BuildCastle(b)
	for y := 2; y <= 4; y++ {
		tp := b.GetBlock(Vec3{19, y, 14})
		if tp != 0 {
			t.Errorf("central tower door should be clear at (19, %d, 14), got %d", y, tp)
		}
	}
}

func TestCastleWallBlocksAtMidHeight(t *testing.T) {
	bg := buildCastleWorld()
	playerY := float32(5.75)
	from := mgl32.Vec3{5, playerY, 5}
	to := mgl32.Vec3{-5, playerY, 5}

	result, _ := CollideStepped(bg, from, to)
	if result.X() < 0.5 {
		t.Errorf("west wall should block movement at mid-height, got x=%f", result.X())
	}
	t.Logf("wall block: from x=5, stopped at x=%f (wall at x=0)", result.X())
}

func TestCastleEastWallBlocksAtMidHeight(t *testing.T) {
	bg := buildCastleWorld()
	playerY := float32(5.75)
	from := mgl32.Vec3{25, playerY, 5}
	to := mgl32.Vec3{35, playerY, 5}

	result, _ := CollideStepped(bg, from, to)
	if result.X() > 29 {
		t.Errorf("east wall should block movement, got x=%f", result.X())
	}
	t.Logf("east wall block: stopped at x=%f (wall at x=29)", result.X())
}

func TestCastleNorthWallBlocksAtMidHeight(t *testing.T) {
	bg := buildCastleWorld()
	playerY := float32(5.75)
	from := mgl32.Vec3{5, playerY, 5}
	to := mgl32.Vec3{5, playerY, -5}

	result, _ := CollideStepped(bg, from, to)
	if result.Z() < 0.5 {
		t.Errorf("north wall should block movement, got z=%f", result.Z())
	}
}

func TestCastleSouthWallBlocksExceptGate(t *testing.T) {
	bg := buildCastleWorld()
	playerY := float32(3.75)

	from := mgl32.Vec3{5, playerY, 25}
	to := mgl32.Vec3{5, playerY, 35}
	result, _ := CollideStepped(bg, from, to)
	if result.Z() > 29 {
		t.Errorf("south wall should block at x=5, got z=%f", result.Z())
	}

	from2 := mgl32.Vec3{14, playerY, 25}
	to2 := mgl32.Vec3{14, playerY, 35}
	result2, _ := CollideStepped(bg, from2, to2)
	if result2.Z() < 30 {
		t.Errorf("gatehouse opening should allow passage at x=14, got z=%f", result2.Z())
	}
	t.Logf("gate passage: z=%f", result2.Z())
}

func TestCastleTowerWallBlocksEntry(t *testing.T) {
	bg := buildCastleWorld()
	playerY := float32(3.75)

	from := mgl32.Vec3{14, playerY, 8}
	to := mgl32.Vec3{14, playerY, 14}
	result, _ := CollideStepped(bg, from, to)

	b := NewStructureBuilder()
	BuildCastle(b)
	wallZ := -1
	for z := 8; z <= 14; z++ {
		if b.GetBlock(Vec3{14, 3, z}) != 0 {
			wallZ = z
			break
		}
	}
	if wallZ >= 0 && result.Z() > float32(wallZ) {
		t.Errorf("tower wall should block entry from north, stopped at z=%f (wall at z=%d)", result.Z(), wallZ)
	}
	t.Logf("tower north wall block: z=%f", result.Z())
}

func TestCastleTowerDoorAllowsEntry(t *testing.T) {
	bg := buildCastleWorld()
	playerY := float32(3.75)

	from := mgl32.Vec3{22, playerY, 14}
	to := mgl32.Vec3{17, playerY, 14}
	result, _ := CollideStepped(bg, from, to)

	if result.X() > 19.5 {
		t.Errorf("tower door should allow entry from east, stopped at x=%f", result.X())
	}
	t.Logf("tower door entry: x=%f", result.X())
}

func TestCastleSpiralStaircaseSteps(t *testing.T) {
	b := NewStructureBuilder()
	BuildCastle(b)
	bg := NewMockBlockGetter()
	b.ApplyTo(bg)

	stairStepCount := 0
	highestStep := 0
	for y := 3; y <= 20; y++ {
		for x := 10; x <= 18; x++ {
			for z := 10; z <= 18; z++ {
				if b.GetBlock(Vec3{x, y, z}) == blockSandstone {
					aboveClear := b.GetBlock(Vec3{x, y + 1, z}) == 0
					if aboveClear {
						stairStepCount++
						if y > highestStep {
							highestStep = y
						}
					}
				}
			}
		}
	}

	if stairStepCount < 10 {
		t.Errorf("staircase should have at least 10 walkable steps, found %d", stairStepCount)
	}
	if highestStep < 15 {
		t.Errorf("staircase should reach at least y=15, highest step at y=%d", highestStep)
	}
	t.Logf("spiral staircase: %d steps, highest at y=%d", stairStepCount, highestStep)
}

func TestCastleSpiralStaircaseStandable(t *testing.T) {
	bg := buildCastleWorld()
	b := NewStructureBuilder()
	BuildCastle(b)

	standableSteps := 0
	for y := 3; y <= 20; y++ {
		for x := 10; x <= 18; x++ {
			for z := 10; z <= 18; z++ {
				if b.GetBlock(Vec3{x, y, z}) == blockSandstone {
					playerPos := mgl32.Vec3{float32(x), float32(y) + 0.6, float32(z)}
					_, stop := CollideStep(bg, playerPos)
					if stop {
						standableSteps++
					}
				}
			}
		}
	}

	if standableSteps < 10 {
		t.Errorf("staircase should have at least 10 standable positions, found %d", standableSteps)
	}
	t.Logf("standable staircase positions: %d", standableSteps)
}

func TestCastleBridgeBlocksExist(t *testing.T) {
	b := NewStructureBuilder()
	BuildCastle(b)

	bridgeY := 9
	bridgeZ := 14
	woodCount := 0
	for x := 19; x <= 29; x++ {
		tp := b.GetBlock(Vec3{x, bridgeY, bridgeZ})
		if tp == blockWood {
			woodCount++
		}
	}
	if woodCount < 8 {
		t.Errorf("bridge should have at least 8 wood blocks, found %d", woodCount)
	}

	railCount := 0
	for x := 19; x <= 29; x++ {
		if b.GetBlock(Vec3{x, bridgeY + 1, bridgeZ - 2}) == blockStoneBrick {
			railCount++
		}
		if b.GetBlock(Vec3{x, bridgeY + 1, bridgeZ + 1}) == blockStoneBrick {
			railCount++
		}
	}
	if railCount < 10 {
		t.Errorf("bridge should have railing blocks, found %d", railCount)
	}
	t.Logf("bridge: %d wood blocks, %d railing blocks", woodCount, railCount)
}

func TestCastleBridgeStandable(t *testing.T) {
	bg := buildCastleWorld()
	bridgeY := 9

	standCount := 0
	for x := 19; x <= 29; x++ {
		for _, z := range []int{13, 14} {
			playerPos := mgl32.Vec3{float32(x), float32(bridgeY) + 0.6, float32(z)}
			_, stop := CollideStep(bg, playerPos)
			if stop {
				standCount++
			}
		}
	}
	if standCount < 15 {
		t.Errorf("should be able to stand on at least 15 bridge positions, got %d", standCount)
	}
	t.Logf("bridge standable positions: %d", standCount)
}

func TestCastleCentralTowerWallsSolid(t *testing.T) {
	b := NewStructureBuilder()
	BuildCastle(b)

	solidCount := 0
	for y := 2; y <= 21; y++ {
		tp := b.GetBlock(Vec3{14 + 5, y, 14})
		if tp != 0 {
			solidCount++
		}
	}
	if solidCount < 10 {
		t.Errorf("central tower east wall should have many solid blocks, got %d", solidCount)
	}
}

func TestCastleCentralTowerTopFloor(t *testing.T) {
	b := NewStructureBuilder()
	BuildCastle(b)

	topFloorY := 21
	solidCount := 0
	for x := 11; x <= 17; x++ {
		for z := 11; z <= 17; z++ {
			if b.GetBlock(Vec3{x, topFloorY, z}) != 0 {
				solidCount++
			}
		}
	}
	if solidCount < 5 {
		t.Errorf("central tower top floor at y=%d should have solid blocks, found %d", topFloorY, solidCount)
	}
	t.Logf("central tower top floor: %d solid blocks", solidCount)
}

func TestCastleCornerTowersExist(t *testing.T) {
	b := NewStructureBuilder()
	BuildCastle(b)

	corners := [][2]int{{0, 0}, {29, 0}, {0, 29}, {29, 29}}
	for _, c := range corners {
		solidCount := 0
		for y := 2; y <= 15; y++ {
			if b.GetBlock(Vec3{c[0], y, c[1]}) != 0 {
				solidCount++
			}
		}
		if solidCount < 5 {
			t.Errorf("corner tower at (%d, %d) has too few solid blocks: %d", c[0], c[1], solidCount)
		}
	}
}

func TestCastleCornerTowerConeRoof(t *testing.T) {
	b := NewStructureBuilder()
	BuildCastle(b)

	roofCount := 0
	for y := 16; y <= 20; y++ {
		for x := -5; x <= 5; x++ {
			for z := -5; z <= 5; z++ {
				if b.GetBlock(Vec3{x, y, z}) == blockBrick {
					roofCount++
				}
			}
		}
	}
	if roofCount < 5 {
		t.Errorf("corner tower at (0,0) should have cone roof, found %d brick blocks", roofCount)
	}
}

func TestCastleBarbican(t *testing.T) {
	b := NewStructureBuilder()
	BuildCastle(b)

	midX := 14
	for y := 2; y <= 4; y++ {
		if b.GetBlock(Vec3{midX - 4, y, 33}) == 0 {
			t.Errorf("barbican west wall missing at (%d, %d, 33)", midX-4, y)
		}
		if b.GetBlock(Vec3{midX + 4, y, 33}) == 0 {
			t.Errorf("barbican east wall missing at (%d, %d, 33)", midX+4, y)
		}
	}

	for x := 12; x <= 16; x++ {
		for y := 2; y <= 4; y++ {
			if b.GetBlock(Vec3{x, y, 35}) != 0 {
				t.Errorf("barbican gate should be open at (%d, %d, 35)", x, y)
			}
		}
	}
}

func TestCastleWindowsGlass(t *testing.T) {
	b := NewStructureBuilder()
	BuildCastle(b)

	glassCount := 0
	for y := 2; y <= 9; y++ {
		for z := 0; z <= 29; z++ {
			if b.GetBlock(Vec3{0, y, z}) == blockGlass {
				glassCount++
			}
			if b.GetBlock(Vec3{29, y, z}) == blockGlass {
				glassCount++
			}
		}
	}
	if glassCount < 4 {
		t.Errorf("castle walls should have glass windows, found %d", glassCount)
	}
	t.Logf("wall glass blocks: %d", glassCount)
}

func TestCastleWallTopWalkway(t *testing.T) {
	b := NewStructureBuilder()
	BuildCastle(b)

	walkwayCount := 0
	for x := 0; x <= 1; x++ {
		for z := 0; z <= 29; z++ {
			if b.GetBlock(Vec3{x, 9, z}) != 0 {
				walkwayCount++
			}
		}
	}
	if walkwayCount < 50 {
		t.Errorf("west wall top walkway should have at least 50 blocks, found %d", walkwayCount)
	}
}

func TestCastleBattlementsExist(t *testing.T) {
	b := NewStructureBuilder()
	BuildCastle(b)

	battlementCount := 0
	for x := 0; x <= 29; x++ {
		if b.GetBlock(Vec3{x, 10, 0}) != 0 {
			battlementCount++
		}
	}
	if battlementCount < 10 {
		t.Errorf("north wall battlements should have at least 10 blocks, found %d", battlementCount)
	}
}

func TestStructureBuilderFillBox(t *testing.T) {
	b := NewStructureBuilder()
	b.FillBox(0, 0, 0, 2, 2, 2, 4)
	if b.BlockCount() != 27 {
		t.Errorf("FillBox 3x3x3 block count = %d, want 27", b.BlockCount())
	}
	for x := 0; x <= 2; x++ {
		for y := 0; y <= 2; y++ {
			for z := 0; z <= 2; z++ {
				if b.GetBlock(Vec3{x, y, z}) != 4 {
					t.Errorf("block at (%d,%d,%d) = %d, want 4", x, y, z, b.GetBlock(Vec3{x, y, z}))
				}
			}
		}
	}
}

func TestStructureBuilderCylinder(t *testing.T) {
	b := NewStructureBuilder()
	b.FillCylinder(10, 10, 3, 0, 0, 4)
	count := b.BlockCount()
	if count < 20 || count > 40 {
		t.Errorf("cylinder r=3 h=1 block count = %d, expected 20-40", count)
	}
	if b.GetBlock(Vec3{10, 0, 10}) != 4 {
		t.Error("cylinder center should be filled")
	}
	if b.GetBlock(Vec3{20, 0, 10}) != 0 {
		t.Error("outside cylinder should be empty")
	}
}

func TestStructureBuilderHollowCylinder(t *testing.T) {
	b := NewStructureBuilder()
	b.FillHollowCylinder(10, 10, 5, 3, 0, 0, 4)
	if b.GetBlock(Vec3{10, 0, 10}) != 0 {
		t.Error("hollow cylinder center should be empty")
	}
	if b.GetBlock(Vec3{15, 0, 10}) != 4 {
		t.Error("hollow cylinder shell should be filled")
	}
}

func TestStructureBuilderApplyTo(t *testing.T) {
	b := NewStructureBuilder()
	b.Set(5, 5, 5, 4)
	b.Set(6, 6, 6, 7)
	bg := NewMockBlockGetter()
	b.ApplyTo(bg)
	if bg.GetBlock(Vec3{5, 5, 5}) != 4 {
		t.Errorf("applied block = %d, want 4", bg.GetBlock(Vec3{5, 5, 5}))
	}
	if bg.GetBlock(Vec3{6, 6, 6}) != 7 {
		t.Errorf("applied block = %d, want 7", bg.GetBlock(Vec3{6, 6, 6}))
	}
}
