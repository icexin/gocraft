package main

import (
	"math"
	"testing"

	"github.com/go-gl/mathgl/mgl32"
)

func TestTextureAtlasValid(t *testing.T) {
	img, rect, err := loadImage("texture.png")
	if err != nil {
		t.Fatal(err)
	}
	if rect.Dx() != 256 || rect.Dy() != 256 {
		t.Errorf("atlas size = %dx%d, want 256x256", rect.Dx(), rect.Dy())
	}
	if len(img) != 256*256*4 {
		t.Errorf("atlas pixel count = %d, want %d", len(img), 256*256*4)
	}

	checkTile := func(idx int, name string) {
		col := idx % 16
		row := idx / 16
		allMagenta := true
		for py := 0; py < 16; py++ {
			for px := 0; px < 16; px++ {
				off := ((row*16+py)*256 + col*16 + px) * 4
				r, g, b := img[off], img[off+1], img[off+2]
				if !(r == 255 && g == 0 && b == 255) {
					allMagenta = false
				}
			}
		}
		if allMagenta {
			t.Errorf("tile %d (%s) is all magenta (empty)", idx, name)
		}
	}

	checkTile(0, "dirt")
	checkTile(1, "sand")
	checkTile(3, "stone")
	checkTile(5, "cobblestone")
	checkTile(7, "brick")
	checkTile(9, "glass")
	checkTile(14, "leaves")
	checkTile(16, "grass_side")
	checkTile(32, "grass_top")
	checkTile(48, "grass_plant")
}

func TestTextureDescCoversAllItems(t *testing.T) {
	LoadTextureDesc()
	for _, id := range availableItems {
		bt := tex.Texture(id)
		if bt == nil {
			t.Errorf("no texture for item %d", id)
		}
	}
}

func TestNearBlockRounding(t *testing.T) {
	tests := []struct {
		pos      mgl32.Vec3
		expected Vec3
	}{
		{mgl32.Vec3{0, 0, 0}, Vec3{0, 0, 0}},
		{mgl32.Vec3{0.4, 0.4, 0.4}, Vec3{0, 0, 0}},
		{mgl32.Vec3{0.5, 0.5, 0.5}, Vec3{1, 1, 1}},
		{mgl32.Vec3{0.6, 0.6, 0.6}, Vec3{1, 1, 1}},
		{mgl32.Vec3{-0.4, -0.4, -0.4}, Vec3{0, 0, 0}},
		{mgl32.Vec3{-0.6, -0.6, -0.6}, Vec3{-1, -1, -1}},
		{mgl32.Vec3{1.5, 2.5, 3.5}, Vec3{2, 3, 4}},
		{mgl32.Vec3{100.1, 200.9, -50.3}, Vec3{100, 201, -50}},
	}
	for _, tt := range tests {
		result := NearBlock(tt.pos)
		if result != tt.expected {
			t.Errorf("NearBlock(%v) = %v, want %v", tt.pos, result, tt.expected)
		}
	}
}

func TestVec3Directions(t *testing.T) {
	v := Vec3{5, 10, 15}
	if v.Left() != (Vec3{4, 10, 15}) {
		t.Errorf("Left() = %v", v.Left())
	}
	if v.Right() != (Vec3{6, 10, 15}) {
		t.Errorf("Right() = %v", v.Right())
	}
	if v.Up() != (Vec3{5, 11, 15}) {
		t.Errorf("Up() = %v", v.Up())
	}
	if v.Down() != (Vec3{5, 9, 15}) {
		t.Errorf("Down() = %v", v.Down())
	}
	if v.Front() != (Vec3{5, 10, 16}) {
		t.Errorf("Front() = %v", v.Front())
	}
	if v.Back() != (Vec3{5, 10, 14}) {
		t.Errorf("Back() = %v", v.Back())
	}
}

func TestChunkidCalculation(t *testing.T) {
	tests := []struct {
		block    Vec3
		expected Vec3
	}{
		{Vec3{0, 0, 0}, Vec3{0, 0, 0}},
		{Vec3{31, 10, 31}, Vec3{0, 0, 0}},
		{Vec3{32, 10, 0}, Vec3{1, 0, 0}},
		{Vec3{-1, 0, -1}, Vec3{-1, 0, -1}},
		{Vec3{-32, 0, -32}, Vec3{-1, 0, -1}},
		{Vec3{-33, 0, 0}, Vec3{-2, 0, 0}},
		{Vec3{63, 0, 63}, Vec3{1, 0, 1}},
		{Vec3{64, 0, 64}, Vec3{2, 0, 2}},
	}
	for _, tt := range tests {
		result := tt.block.Chunkid()
		if result != tt.expected {
			t.Errorf("Vec3%v.Chunkid() = %v, want %v", tt.block, result, tt.expected)
		}
	}
}

func TestChunkidAlwaysZeroY(t *testing.T) {
	for y := -100; y <= 100; y++ {
		v := Vec3{10, y, 20}
		cid := v.Chunkid()
		if cid.Y != 0 {
			t.Errorf("Chunkid Y should always be 0, got %d for Y=%d", cid.Y, y)
		}
	}
}

func TestCameraCreation(t *testing.T) {
	c := NewCamera(mgl32.Vec3{10, 20, 30})
	pos := c.Pos()
	if pos.X() != 10 || pos.Y() != 20 || pos.Z() != 30 {
		t.Errorf("initial position = %v, want {10, 20, 30}", pos)
	}
	if c.Flying() {
		t.Error("camera should not start flying")
	}
}

func TestCameraFlipFlying(t *testing.T) {
	c := NewCamera(mgl32.Vec3{0, 0, 0})
	if c.Flying() {
		t.Error("should start grounded")
	}
	c.FlipFlying()
	if !c.Flying() {
		t.Error("should be flying after flip")
	}
	c.FlipFlying()
	if c.Flying() {
		t.Error("should be grounded after double flip")
	}
}

func TestCameraMoveForward(t *testing.T) {
	c := NewCamera(mgl32.Vec3{0, 0, 0})
	startPos := c.Pos()
	c.OnMoveChange(MoveForward, 1.0)
	endPos := c.Pos()
	dist := endPos.Sub(startPos).Len()
	if dist < 0.5 {
		t.Errorf("forward movement too small: %f", dist)
	}
}

func TestCameraMoveBackward(t *testing.T) {
	c := NewCamera(mgl32.Vec3{0, 0, 0})
	c.OnMoveChange(MoveForward, 1.0)
	forwardPos := c.Pos()
	c.OnMoveChange(MoveBackward, 1.0)
	backPos := c.Pos()
	dist := backPos.Sub(mgl32.Vec3{0, 0, 0}).Len()
	if dist > 0.01 {
		t.Errorf("forward+backward should return near origin, dist=%f, forwardPos=%v, backPos=%v", dist, forwardPos, backPos)
	}
}

func TestCameraFlyingSpeedMultiplier(t *testing.T) {
	ground := NewCamera(mgl32.Vec3{0, 0, 0})
	ground.OnMoveChange(MoveForward, 0.2)
	groundDist := ground.Pos().Sub(mgl32.Vec3{0, 0, 0}).Len()

	fly := NewCamera(mgl32.Vec3{0, 0, 0})
	fly.FlipFlying()
	fly.OnMoveChange(MoveForward, 0.2)
	flyDist := fly.Pos().Sub(mgl32.Vec3{0, 0, 0}).Len()

	ratio := flyDist / groundDist
	if ratio < 4.5 || ratio > 5.5 {
		t.Errorf("flying speed ratio = %f, want ~5.0 (ground=%f, fly=%f)", ratio, groundDist, flyDist)
	}
}

func TestCameraAngleClamping(t *testing.T) {
	c := NewCamera(mgl32.Vec3{0, 0, 0})
	c.Sens = 1.0
	c.OnAngleChange(0, 100)
	if c.rotatey > 89 {
		t.Errorf("rotatey should be clamped to 89, got %f", c.rotatey)
	}
	c.OnAngleChange(0, -200)
	if c.rotatey < -89 {
		t.Errorf("rotatey should be clamped to -89, got %f", c.rotatey)
	}
}

func TestCameraAngleIgnoresLargeJumps(t *testing.T) {
	c := NewCamera(mgl32.Vec3{0, 0, 0})
	rx, ry := c.rotatex, c.rotatey
	c.OnAngleChange(300, 0)
	if c.rotatex != rx {
		t.Errorf("large dx should be ignored, rotatex changed from %f to %f", rx, c.rotatex)
	}
	c.OnAngleChange(0, 300)
	if c.rotatey != ry {
		t.Errorf("large dy should be ignored, rotatey changed from %f to %f", ry, c.rotatey)
	}
}

func TestCameraState(t *testing.T) {
	c := NewCamera(mgl32.Vec3{1, 2, 3})
	state := c.State()
	if state.X != 1 || state.Y != 2 || state.Z != 3 {
		t.Errorf("state position = (%f,%f,%f), want (1,2,3)", state.X, state.Y, state.Z)
	}
}

func TestCameraRestore(t *testing.T) {
	c := NewCamera(mgl32.Vec3{0, 0, 0})
	state := PlayerState{X: 10, Y: 20, Z: 30, Rx: 45, Ry: -30}
	c.Restore(state)
	pos := c.Pos()
	if pos.X() != 10 || pos.Y() != 20 || pos.Z() != 30 {
		t.Errorf("restored position = %v, want {10,20,30}", pos)
	}
	if c.rotatex != 45 || c.rotatey != -30 {
		t.Errorf("restored rotation = (%f,%f), want (45,-30)", c.rotatex, c.rotatey)
	}
}

func TestCameraMatrixIsValid(t *testing.T) {
	c := NewCamera(mgl32.Vec3{0, 10, 0})
	mat := c.Matrix()
	det := mat.Det()
	if math.IsNaN(float64(det)) || math.IsInf(float64(det), 0) || abs(det) < 0.001 {
		t.Errorf("camera matrix determinant is invalid: %f", det)
	}
}

func TestCameraMoveSideways(t *testing.T) {
	c := NewCamera(mgl32.Vec3{0, 0, 0})
	c.OnMoveChange(MoveLeft, 1.0)
	leftPos := c.Pos()
	c.OnMoveChange(MoveRight, 1.0)
	backPos := c.Pos()
	dist := backPos.Sub(mgl32.Vec3{0, 0, 0}).Len()
	if dist > 0.01 {
		t.Errorf("left+right should return near origin, dist=%f, leftPos=%v", dist, leftPos)
	}
}

func TestHitTestFindsBlock(t *testing.T) {
	bg := NewMockBlockGetter()
	bg.SetBlock(Vec3{0, 0, -3}, 4)

	w := &mockWorld{bg}
	pos := mgl32.Vec3{0, 0, 0}
	dir := mgl32.Vec3{0, 0, -1}

	block, prev := w.HitTest(pos, dir)
	if block == nil {
		t.Fatal("expected to hit block at {0,0,-3}")
	}
	if *block != (Vec3{0, 0, -3}) {
		t.Errorf("hit block = %v, want {0,0,-3}", *block)
	}
	if prev == nil {
		t.Error("expected prev block to be non-nil")
	}
}

func TestHitTestMissesEmptySpace(t *testing.T) {
	bg := NewMockBlockGetter()
	w := &mockWorld{bg}
	pos := mgl32.Vec3{0, 0, 0}
	dir := mgl32.Vec3{0, 0, -1}

	block, _ := w.HitTest(pos, dir)
	if block != nil {
		t.Errorf("expected no hit in empty space, got %v", *block)
	}
}

func TestHitTestMaxRange(t *testing.T) {
	bg := NewMockBlockGetter()
	bg.SetBlock(Vec3{0, 0, -20}, 4)
	w := &mockWorld{bg}
	pos := mgl32.Vec3{0, 0, 0}
	dir := mgl32.Vec3{0, 0, -1}

	block, _ := w.HitTest(pos, dir)
	if block != nil {
		t.Error("block at distance 20 should be out of range (max 8)")
	}
}

func TestChunkAddDelBlock(t *testing.T) {
	cid := Vec3{0, 0, 0}
	c := NewChunk(cid)
	bid := Vec3{5, 10, 15}
	c.add(bid, 4)
	if c.Block(bid) != 4 {
		t.Errorf("Block after add = %d, want 4", c.Block(bid))
	}
	c.del(bid)
	if c.Block(bid) != 0 {
		t.Errorf("Block after del = %d, want 0", c.Block(bid))
	}
}

func TestChunkRangeBlocks(t *testing.T) {
	cid := Vec3{0, 0, 0}
	c := NewChunk(cid)
	c.add(Vec3{1, 0, 0}, 4)
	c.add(Vec3{2, 0, 0}, 5)
	c.add(Vec3{3, 0, 0}, 6)

	count := 0
	c.RangeBlocks(func(id Vec3, w int) {
		count++
		if w < 4 || w > 6 {
			t.Errorf("unexpected block type %d at %v", w, id)
		}
	})
	if count != 3 {
		t.Errorf("RangeBlocks visited %d blocks, want 3", count)
	}
}

type mockWorld struct {
	bg *MockBlockGetter
}

func (w *mockWorld) HasBlock(id Vec3) bool {
	tp := w.bg.GetBlock(id)
	return tp != -1 && tp != 0
}

func (w *mockWorld) HitTest(pos mgl32.Vec3, vec mgl32.Vec3) (*Vec3, *Vec3) {
	var (
		maxLen = float32(8.0)
		step   = float32(0.125)

		block, prev Vec3
		pprev       *Vec3
	)

	for l := float32(0); l < maxLen; l += step {
		block = NearBlock(pos.Add(vec.Mul(l)))
		if prev != block && w.HasBlock(block) {
			return &block, pprev
		}
		prev = block
		pprev = &prev
	}
	return nil, nil
}
