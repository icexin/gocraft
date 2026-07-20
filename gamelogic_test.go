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

func TestFullWireFrameEdgeCount(t *testing.T) {
	var vertices []float32
	vertices = makeFullWireFrameData(vertices)
	vertexCount := len(vertices) / 3
	if vertexCount != 24 {
		t.Errorf("full wireframe should have 24 vertices (12 edges * 2), got %d", vertexCount)
	}
	edgeCount := vertexCount / 2
	if edgeCount != 12 {
		t.Errorf("full wireframe should have 12 edges, got %d", edgeCount)
	}
}

func TestFullWireFrameSymmetry(t *testing.T) {
	var vertices []float32
	vertices = makeFullWireFrameData(vertices)
	for i := 0; i < len(vertices); i += 3 {
		x, y, z := vertices[i], vertices[i+1], vertices[i+2]
		if abs(x) != 0.5 || abs(y) != 0.5 || abs(z) != 0.5 {
			t.Errorf("vertex %d: (%.1f, %.1f, %.1f) should have all components at +/-0.5", i/3, x, y, z)
		}
	}
}

func TestWireFrameDataStillWorks(t *testing.T) {
	show := [6]bool{true, true, true, true, true, true}
	var vertices []float32
	vertices = makeWireFrameData(vertices, show)
	vertexCount := len(vertices) / 3
	if vertexCount != 48 {
		t.Errorf("all-faces wireframe should have 48 vertices (6 faces * 8), got %d", vertexCount)
	}
}

func TestHitTestReturnsPlacementPosition(t *testing.T) {
	bg := NewMockBlockGetter()
	bg.SetBlock(Vec3{0, 0, -3}, 4)

	w := &mockWorld{bg}
	pos := mgl32.Vec3{0, 0, 0}
	dir := mgl32.Vec3{0, 0, -1}

	block, prev := w.HitTest(pos, dir)
	if block == nil {
		t.Fatal("expected to hit block")
	}
	if prev == nil {
		t.Fatal("expected prev (placement) position")
	}
	if *prev == *block {
		t.Errorf("placement position should differ from hit block")
	}
	if prev.Z <= block.Z {
		t.Errorf("placement Z=%d should be between camera and block Z=%d", prev.Z, block.Z)
	}
}

func TestSprintSpeedMultiplier(t *testing.T) {
	c := NewCamera(mgl32.Vec3{0, 0, 0})
	c.SetMovementMode(ModeWalk)
	c.OnMoveChange(MoveForward, 1.0)
	walkDist := c.Pos().Sub(mgl32.Vec3{0, 0, 0}).Len()

	c2 := NewCamera(mgl32.Vec3{0, 0, 0})
	c2.SetMovementMode(ModeSprint)
	c2.OnMoveChange(MoveForward, 1.0)
	sprintDist := c2.Pos().Sub(mgl32.Vec3{0, 0, 0}).Len()

	ratio := sprintDist / walkDist
	if ratio < 1.5 || ratio > 1.7 {
		t.Errorf("sprint/walk ratio = %f, want ~1.6", ratio)
	}
}

func TestCrouchSpeedMultiplier(t *testing.T) {
	c := NewCamera(mgl32.Vec3{0, 0, 0})
	c.SetMovementMode(ModeWalk)
	c.OnMoveChange(MoveForward, 1.0)
	walkDist := c.Pos().Sub(mgl32.Vec3{0, 0, 0}).Len()

	c2 := NewCamera(mgl32.Vec3{0, 0, 0})
	c2.SetMovementMode(ModeCrouch)
	c2.OnMoveChange(MoveForward, 1.0)
	crouchDist := c2.Pos().Sub(mgl32.Vec3{0, 0, 0}).Len()

	ratio := crouchDist / walkDist
	if ratio < 0.45 || ratio > 0.55 {
		t.Errorf("crouch/walk ratio = %f, want ~0.5", ratio)
	}
}

func TestFOVInterpolation(t *testing.T) {
	c := NewCamera(mgl32.Vec3{0, 0, 0})
	if c.CurrentFOV() != 65 {
		t.Errorf("initial FOV = %f, want 65", c.CurrentFOV())
	}
	c.SetMovementMode(ModeSprint)
	for i := 0; i < 100; i++ {
		c.UpdateFOV(0.016)
	}
	if c.CurrentFOV() < 74 || c.CurrentFOV() > 76 {
		t.Errorf("sprint FOV after 100 ticks = %f, want ~75", c.CurrentFOV())
	}
	c.SetMovementMode(ModeWalk)
	for i := 0; i < 100; i++ {
		c.UpdateFOV(0.016)
	}
	if c.CurrentFOV() < 64 || c.CurrentFOV() > 66 {
		t.Errorf("walk FOV after 100 ticks = %f, want ~65", c.CurrentFOV())
	}
}

func TestCrouchOffset(t *testing.T) {
	c := NewCamera(mgl32.Vec3{0, 0, 0})
	c.SetMovementMode(ModeCrouch)
	for i := 0; i < 100; i++ {
		c.UpdateFOV(0.016)
	}
	if c.CrouchOffset() > -0.25 || c.CrouchOffset() < -0.35 {
		t.Errorf("crouch offset = %f, want ~-0.3", c.CrouchOffset())
	}
	c.SetMovementMode(ModeWalk)
	for i := 0; i < 100; i++ {
		c.UpdateFOV(0.016)
	}
	if abs(c.CrouchOffset()) > 0.05 {
		t.Errorf("walk offset = %f, want ~0", c.CrouchOffset())
	}
}

func TestSprintRequiresForwardMovement(t *testing.T) {
	c := NewCamera(mgl32.Vec3{0, 0, 0})
	c.SetMovementMode(ModeSprint)
	mult := c.SpeedMultiplier()
	if mult < 1.5 || mult > 1.7 {
		t.Errorf("sprint multiplier = %f, want ~1.6", mult)
	}
	c.SetMovementMode(ModeWalk)
	mult = c.SpeedMultiplier()
	if mult != 1.0 {
		t.Errorf("walk multiplier = %f, want 1.0", mult)
	}
}

func TestSprintFlyingMultiplier(t *testing.T) {
	c := NewCamera(mgl32.Vec3{0, 0, 0})
	c.FlipFlying()
	c.SetMovementMode(ModeSprint)
	mult := c.SpeedMultiplier()
	if mult < 7.5 || mult > 8.5 {
		t.Errorf("flying sprint multiplier = %f, want ~8.0", mult)
	}
}

func TestEditHistoryPushPop(t *testing.T) {
	h := NewEditHistory(50)
	h.Push(EditAction{Position: Vec3{1, 2, 3}, OldType: 4, NewType: 0})
	h.Push(EditAction{Position: Vec3{4, 5, 6}, OldType: 0, NewType: 7})
	if h.Len() != 2 {
		t.Errorf("history length = %d, want 2", h.Len())
	}
	action, ok := h.Pop()
	if !ok {
		t.Fatal("expected pop to succeed")
	}
	if action.Position != (Vec3{4, 5, 6}) || action.OldType != 0 || action.NewType != 7 {
		t.Errorf("popped action = %+v, want {4,5,6} old=0 new=7", action)
	}
	action, ok = h.Pop()
	if !ok {
		t.Fatal("expected second pop to succeed")
	}
	if action.Position != (Vec3{1, 2, 3}) {
		t.Errorf("second popped position = %v, want {1,2,3}", action.Position)
	}
	_, ok = h.Pop()
	if ok {
		t.Error("pop from empty history should return false")
	}
}

func TestEditHistoryOverflow(t *testing.T) {
	h := NewEditHistory(5)
	for i := 0; i < 10; i++ {
		h.Push(EditAction{Position: Vec3{i, 0, 0}, OldType: 0, NewType: i + 1})
	}
	if h.Len() != 5 {
		t.Errorf("history length = %d, want 5 (capped)", h.Len())
	}
	action, _ := h.Pop()
	if action.Position.X != 9 {
		t.Errorf("most recent action x = %d, want 9", action.Position.X)
	}
	action, _ = h.Pop()
	if action.Position.X != 8 {
		t.Errorf("second most recent action x = %d, want 8", action.Position.X)
	}
}

func TestEditHistoryClear(t *testing.T) {
	h := NewEditHistory(50)
	h.Push(EditAction{Position: Vec3{1, 2, 3}, OldType: 4, NewType: 0})
	h.Push(EditAction{Position: Vec3{4, 5, 6}, OldType: 0, NewType: 7})
	h.Clear()
	if h.Len() != 0 {
		t.Errorf("history length after clear = %d, want 0", h.Len())
	}
	_, ok := h.Pop()
	if ok {
		t.Error("pop after clear should return false")
	}
}

func TestUndoRestoresBlock(t *testing.T) {
	bg := NewMockBlockGetter()
	bg.SetBlock(Vec3{5, 10, 5}, 4)

	h := NewEditHistory(50)
	oldType := bg.GetBlock(Vec3{5, 10, 5})
	h.Push(EditAction{Position: Vec3{5, 10, 5}, OldType: oldType, NewType: 0})
	bg.SetBlock(Vec3{5, 10, 5}, 0)
	if bg.GetBlock(Vec3{5, 10, 5}) != 0 {
		t.Fatal("block should be air after break")
	}

	action, ok := h.Pop()
	if !ok {
		t.Fatal("expected pop to succeed")
	}
	bg.SetBlock(action.Position, action.OldType)
	if bg.GetBlock(Vec3{5, 10, 5}) != 4 {
		t.Errorf("block after undo = %d, want 4", bg.GetBlock(Vec3{5, 10, 5}))
	}
}

func TestWriteQueueBatching(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	s.queue.Enqueue(Vec3{1, 2, 3}, 4)
	s.queue.Enqueue(Vec3{4, 5, 6}, 7)
	s.queue.Enqueue(Vec3{7, 8, 9}, 10)
	flushed := s.queue.Flush()
	if flushed != 3 {
		t.Errorf("flushed = %d, want 3", flushed)
	}

	var count int
	s.RangeBlocks(Vec3{0, 0, 0}, func(bid Vec3, w int) {
		count++
	})
	if count != 3 {
		t.Errorf("stored blocks = %d, want 3", count)
	}
}

func TestWriteQueueFlushOnClose(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir + "/test.db")
	if err != nil {
		t.Fatal(err)
	}

	s.queue.Enqueue(Vec3{10, 20, 30}, 5)
	s.Close()

	s2, err := NewStore(dir + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()

	var found bool
	s2.RangeBlocks(Vec3{0, 0, 0}, func(bid Vec3, w int) {
		if bid == (Vec3{10, 20, 30}) && w == 5 {
			found = true
		}
	})
	if !found {
		t.Error("block should persist after close")
	}
}

func TestGameClockTimeOfDay(t *testing.T) {
	c := NewGameClock(600)
	if c.TimeOfDay(0) != 0 {
		t.Errorf("TimeOfDay(0) = %f, want 0", c.TimeOfDay(0))
	}
	if c.TimeOfDay(300) != 0.5 {
		t.Errorf("TimeOfDay(300) = %f, want 0.5", c.TimeOfDay(300))
	}
	if c.TimeOfDay(600) != 0 {
		t.Errorf("TimeOfDay(600) = %f, want 0 (wrap)", c.TimeOfDay(600))
	}
	if c.TimeOfDay(750) != 0.25 {
		t.Errorf("TimeOfDay(750) = %f, want 0.25", c.TimeOfDay(750))
	}
	c.SetOffset(100)
	if c.TimeOfDay(100) != 0 {
		t.Errorf("TimeOfDay with offset: %f, want 0", c.TimeOfDay(100))
	}
}

func TestGameClockSkyColorInterpolation(t *testing.T) {
	noon := skyColorAtTime(0.5)
	if noon.X() < 0.5 || noon.Y() < 0.5 {
		t.Errorf("noon sky too dark: %v", noon)
	}
	midnight := skyColorAtTime(0.0)
	if midnight.X() > 0.1 || midnight.Y() > 0.1 {
		t.Errorf("midnight sky too bright: %v", midnight)
	}
	sunrise := skyColorAtTime(0.25)
	if sunrise.X() < 0.2 {
		t.Errorf("sunrise sky should have warm red component: %v", sunrise)
	}
	sunset := skyColorAtTime(0.75)
	if sunset.X() < 0.2 {
		t.Errorf("sunset sky should have warm red component: %v", sunset)
	}
}

func TestGameClockSunDirection(t *testing.T) {
	c := NewGameClock(600)
	c.SetOffset(0)

	noonDir := sunDirAtTime(c, 0.5)
	if noonDir.Y() < 0.5 {
		t.Errorf("noon sun should point upward, Y=%f", noonDir.Y())
	}

	midnightDir := sunDirAtTime(c, 0.0)
	if midnightDir.Y() > -0.5 {
		t.Errorf("midnight sun should point downward, Y=%f", midnightDir.Y())
	}
}

func sunDirAtTime(c *GameClock, tod float64) mgl32.Vec3 {
	angle := (tod - 0.25) * 2 * math.Pi
	return mgl32.Vec3{
		-float32(math.Cos(angle)),
		float32(math.Sin(angle)),
		-0.3,
	}.Normalize()
}

func TestGameClockAmbientLevel(t *testing.T) {
	noon := ambientAtTime(0.5)
	if noon != 0.5 {
		t.Errorf("noon ambient = %f, want 0.5", noon)
	}
	midnight := ambientAtTime(0.0)
	if midnight != 0.15 {
		t.Errorf("midnight ambient = %f, want 0.15", midnight)
	}
	sunrise := ambientAtTime(0.25)
	if sunrise < 0.25 || sunrise > 0.4 {
		t.Errorf("sunrise ambient = %f, want between 0.25 and 0.4", sunrise)
	}
}

func TestHotbarSlotSelection(t *testing.T) {
	h := NewHotbar()
	if h.Selected() != 0 {
		t.Errorf("initial slot = %d, want 0", h.Selected())
	}
	if h.SelectedItem() != availableItems[0] {
		t.Errorf("initial item = %d, want %d", h.SelectedItem(), availableItems[0])
	}
	h.Select(4)
	if h.Selected() != 4 {
		t.Errorf("after Select(4), slot = %d", h.Selected())
	}
	if h.SelectedItem() != availableItems[4] {
		t.Errorf("slot 4 item = %d, want %d", h.SelectedItem(), availableItems[4])
	}
	h.NextSlot()
	if h.Selected() != 5 {
		t.Errorf("after NextSlot, slot = %d, want 5", h.Selected())
	}
	h.Select(8)
	h.NextSlot()
	if h.Selected() != 0 {
		t.Errorf("NextSlot from 8 should wrap to 0, got %d", h.Selected())
	}
	h.PrevSlot()
	if h.Selected() != 8 {
		t.Errorf("PrevSlot from 0 should wrap to 8, got %d", h.Selected())
	}
}

func TestHotbarSetSlot(t *testing.T) {
	h := NewHotbar()
	h.SetSlot(3, 10)
	if h.GetSlot(3) != 10 {
		t.Errorf("slot 3 = %d, want 10", h.GetSlot(3))
	}
	h.Select(3)
	if h.SelectedItem() != 10 {
		t.Errorf("selected item after SetSlot = %d, want 10", h.SelectedItem())
	}
}

func TestBitmapFontUVMapping(t *testing.T) {
	u, v, u2, v2 := GlyphUV(' ')
	if u != 0 || v != 0 {
		t.Errorf("space UV start = (%f, %f), want (0, 0)", u, v)
	}
	if u2 <= u || v2 <= v {
		t.Errorf("space UV end should be > start: (%f, %f)", u2, v2)
	}

	u, v, _, _ = GlyphUV('!')
	expectedU := float32(FontGlyphSize) / float32(FontAtlasCols*FontGlyphSize)
	if abs(u-expectedU) > 0.001 {
		t.Errorf("'!' U = %f, want %f", u, expectedU)
	}
	if v != 0 {
		t.Errorf("'!' V = %f, want 0 (first row)", v)
	}

	u, v, _, _ = GlyphUV('0')
	idx := int('0' - FontStartChar)
	col := idx % FontAtlasCols
	row := idx / FontAtlasCols
	wantU := float32(col*FontGlyphSize) / float32(FontAtlasCols*FontGlyphSize)
	wantV := float32(row*FontGlyphSize) / float32(FontAtlasRows*FontGlyphSize)
	if abs(u-wantU) > 0.001 || abs(v-wantV) > 0.001 {
		t.Errorf("'0' UV = (%f, %f), want (%f, %f)", u, v, wantU, wantV)
	}
}

func TestFormatDebugOverlay(t *testing.T) {
	state := HUDState{
		FPS:        60,
		PlayerPos:  [3]float32{10.5, 20.3, -30.1},
		ChunkID:    Vec3{1, 0, -1},
		ChunkCount: 100,
		FaceCount:  5000,
	}
	s := FormatDebugOverlay(state)
	if len(s) == 0 {
		t.Error("debug overlay should not be empty")
	}
}

func TestBlockMaterialMapping(t *testing.T) {
	for tp := 1; tp <= 16; tp++ {
		mat := BlockMaterialFromType(tp)
		if mat < MaterialStone || mat > MaterialGravel {
			t.Errorf("block type %d mapped to invalid material %d", tp, mat)
		}
	}
	if BlockMaterialFromType(2) != MaterialSand {
		t.Error("sand block should map to MaterialSand")
	}
	if BlockMaterialFromType(5) != MaterialWood {
		t.Error("wood block should map to MaterialWood")
	}
	if BlockMaterialFromType(10) != MaterialGlass {
		t.Error("glass block should map to MaterialGlass")
	}
}

func TestFootstepSoundWaveform(t *testing.T) {
	materials := []BlockMaterial{MaterialStone, MaterialDirt, MaterialWood, MaterialSand, MaterialGlass, MaterialGravel}
	for _, mat := range materials {
		gen := NewFootstepSound(mat)
		hasNonZero := false
		for i := 0; i < 100; i++ {
			s := gen.Sample(float64(i) * 0.001)
			if s != 0 {
				hasNonZero = true
			}
			if s > 1.0 || s < -1.0 {
				t.Errorf("material %d: sample out of range at t=%f: %f", mat, float64(i)*0.001, s)
			}
		}
		if !hasNonZero {
			t.Errorf("material %d: footstep produced all-zero samples", mat)
		}
	}
}

func TestMixerMultipleVoices(t *testing.T) {
	m := NewMixer()
	m.Play(NewFootstepSound(MaterialStone), 0.08, 0.5)
	m.Play(NewFootstepSound(MaterialWood), 0.08, 0.5)
	if m.ActiveVoices() != 2 {
		t.Errorf("active voices = %d, want 2", m.ActiveVoices())
	}
	samples := m.ReadSamples(100)
	if len(samples) != 100 {
		t.Errorf("sample count = %d, want 100", len(samples))
	}
	hasNonZero := false
	for _, s := range samples {
		if s != 0 {
			hasNonZero = true
		}
	}
	if !hasNonZero {
		t.Error("mixed samples should not be all zero")
	}
}

func TestSoundDuration(t *testing.T) {
	m := NewMixer()
	m.Play(NewFootstepSound(MaterialStone), 0.001, 0.5)
	if m.ActiveVoices() != 1 {
		t.Errorf("initial active voices = %d, want 1", m.ActiveVoices())
	}
	m.ReadSamples(SampleRate)
	if m.ActiveVoices() != 0 {
		t.Errorf("after 1s of samples, active voices = %d, want 0", m.ActiveVoices())
	}
}

func TestBlockBreakSoundWaveform(t *testing.T) {
	gen := NewBlockBreakSound(MaterialStone)
	hasNonZero := false
	for i := 0; i < 200; i++ {
		s := gen.Sample(float64(i) * 0.001)
		if s != 0 {
			hasNonZero = true
		}
	}
	if !hasNonZero {
		t.Error("block break should produce non-zero samples")
	}
}

func TestBlockPlaceSoundWaveform(t *testing.T) {
	gen := NewBlockPlaceSound(MaterialWood)
	hasNonZero := false
	for i := 0; i < 150; i++ {
		s := gen.Sample(float64(i) * 0.001)
		if s != 0 {
			hasNonZero = true
		}
	}
	if !hasNonZero {
		t.Error("block place should produce non-zero samples")
	}
}

func TestAdjacentChunkDirty(t *testing.T) {
	blockAtBoundary := Vec3{32, 5, 16}
	cid := blockAtBoundary.Chunkid()
	if cid != (Vec3{1, 0, 0}) {
		t.Fatalf("expected chunk {1,0,0}, got %v", cid)
	}
	neighbor := blockAtBoundary.Left()
	neighborCid := neighbor.Chunkid()
	if neighborCid == cid {
		t.Error("left neighbor should be in a different chunk")
	}
	if neighborCid != (Vec3{0, 0, 0}) {
		t.Errorf("left neighbor chunk = %v, want {0,0,0}", neighborCid)
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
