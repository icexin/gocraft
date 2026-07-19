package main

import (
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/go-gl/mathgl/mgl32"
)

type MockBlockGetter struct {
	blocks map[Vec3]int
}

func NewMockBlockGetter() *MockBlockGetter {
	return &MockBlockGetter{blocks: make(map[Vec3]int)}
}

func (m *MockBlockGetter) SetBlock(v Vec3, tp int) {
	m.blocks[v] = tp
}

func (m *MockBlockGetter) GetBlock(id Vec3) int {
	if tp, ok := m.blocks[id]; ok {
		return tp
	}
	return 0
}

func (m *MockBlockGetter) FillFloor(y int, xmin, xmax, zmin, zmax int) {
	for x := xmin; x <= xmax; x++ {
		for z := zmin; z <= zmax; z++ {
			m.SetBlock(Vec3{x, y, z}, 4)
		}
	}
}

func (m *MockBlockGetter) FillWall(x, ymin, ymax, zmin, zmax int) {
	for y := ymin; y <= ymax; y++ {
		for z := zmin; z <= zmax; z++ {
			m.SetBlock(Vec3{x, y, z}, 4)
		}
	}
}

func TestCollideStepStandingOnFloor(t *testing.T) {
	bg := NewMockBlockGetter()
	bg.FillFloor(0, -5, 5, -5, 5)

	pos := mgl32.Vec3{0, 0.6, 0}
	result, stop := CollideStep(bg, pos)

	if !stop {
		t.Error("should stop when near floor")
	}
	if result.Y() < 0.75 {
		t.Errorf("y should be at least 0.75 (pad above floor), got %f", result.Y())
	}
	t.Logf("standing on floor: y=%f, stop=%v", result.Y(), stop)
}

func TestCollideStepHighAboveFloor(t *testing.T) {
	bg := NewMockBlockGetter()
	bg.FillFloor(0, -5, 5, -5, 5)

	pos := mgl32.Vec3{0, 5, 0}
	result, stop := CollideStep(bg, pos)

	if stop {
		t.Error("should not stop high above floor")
	}
	if result.Y() != 5 {
		t.Errorf("y should be unchanged at 5, got %f", result.Y())
	}
}

func TestCollideStepWallBlocksApproach(t *testing.T) {
	bg := NewMockBlockGetter()
	bg.FillWall(2, 0, 3, -5, 5)

	pos := mgl32.Vec3{1.3, 1, 0}
	result, _ := CollideStep(bg, pos)

	if result.X() > 1.75 {
		t.Errorf("approaching wall at x=2 from x=1.3: should clamp to 1.75, got x=%f", result.X())
	}
	t.Logf("wall approach: x=%f", result.X())
}

func TestCollideStepNoObstacle(t *testing.T) {
	bg := NewMockBlockGetter()

	pos := mgl32.Vec3{5.5, 10.5, 3.5}
	result, stop := CollideStep(bg, pos)

	if stop {
		t.Error("should not stop in empty space")
	}
	if result != pos {
		t.Errorf("position should be unchanged in empty space, got %v", result)
	}
}

func TestCollideStepPlantNotObstacle(t *testing.T) {
	bg := NewMockBlockGetter()
	bg.SetBlock(Vec3{2, 1, 0}, 17)
	bg.SetBlock(Vec3{2, 0, 0}, 17)

	pos := mgl32.Vec3{1.8, 1, 0}
	result, _ := CollideStep(bg, pos)

	if result.X() != 1.8 {
		t.Errorf("plants should not block, expected x=1.8, got x=%f", result.X())
	}
}

func TestCollideSteppedBlocksAtWall(t *testing.T) {
	bg := NewMockBlockGetter()
	bg.FillWall(5, 0, 10, -10, 10)

	from := mgl32.Vec3{3, 5, 0}
	to := mgl32.Vec3{10, 5, 0}

	result, _ := CollideStepped(bg, from, to)

	if result.X() > 4.8 {
		t.Errorf("stepped collision should stop before wall at x=5, got x=%f", result.X())
	}
	t.Logf("stepped collision: from x=3 to x=10, stopped at x=%f", result.X())
}

func TestCollideSteppedNoWall(t *testing.T) {
	bg := NewMockBlockGetter()

	from := mgl32.Vec3{0, 5, 0}
	to := mgl32.Vec3{10, 5, 0}

	result, _ := CollideStepped(bg, from, to)

	diff := abs(result.X() - 10)
	if diff > 0.01 {
		t.Errorf("no wall: expected x~10, got x=%f (diff=%f)", result.X(), diff)
	}
}

func TestCollideSteppedSmallMovement(t *testing.T) {
	bg := NewMockBlockGetter()

	from := mgl32.Vec3{0, 2, 0}
	to := mgl32.Vec3{0.05, 2, 0}

	result, _ := CollideStepped(bg, from, to)

	diff := result.Sub(to).Len()
	if diff > 0.001 {
		t.Errorf("small movement should pass through unmodified, diff=%f", diff)
	}
}

func TestCollideSteppedFlyingIntoWall(t *testing.T) {
	bg := NewMockBlockGetter()
	bg.FillWall(3, 0, 10, -10, 10)

	from := mgl32.Vec3{1, 5, 0}
	to := mgl32.Vec3{5, 5, 0}

	result, _ := CollideStepped(bg, from, to)

	if result.X() > 2.8 {
		t.Errorf("flying into wall at x=3: should stop before it, got x=%f", result.X())
	}
	t.Logf("flying into wall: stopped at x=%f (wall at x=3)", result.X())
}

func TestCollideSteppedZeroMovement(t *testing.T) {
	bg := NewMockBlockGetter()
	pos := mgl32.Vec3{5, 5, 5}

	result, _ := CollideStepped(bg, pos, pos)

	if result != pos {
		t.Errorf("zero movement: position should be unchanged, got %v", result)
	}
}

func TestIsObstacleClassification(t *testing.T) {
	tests := []struct {
		tp       int
		expected bool
	}{
		{-1, true},
		{0, false},
		{1, true},
		{4, true},
		{10, true},
		{15, true},
		{17, false},
		{23, false},
		{31, false},
		{32, true},
	}
	for _, tt := range tests {
		result := IsObstacle(tt.tp)
		if result != tt.expected {
			t.Errorf("IsObstacle(%d) = %v, want %v", tt.tp, result, tt.expected)
		}
	}
}

func TestIsPlantClassification(t *testing.T) {
	for tp := 17; tp <= 31; tp++ {
		if !IsPlant(tp) {
			t.Errorf("IsPlant(%d) should be true", tp)
		}
	}
	nonPlants := []int{0, 1, 4, 16, 32}
	for _, tp := range nonPlants {
		if IsPlant(tp) {
			t.Errorf("IsPlant(%d) should be false", tp)
		}
	}
}

func TestSkyColorDefault(t *testing.T) {
	*importedWorld = false
	sky := SkyColor()
	if sky.X() < 0.5 || sky.Y() < 0.5 || sky.Z() < 0.5 {
		t.Errorf("default sky should be bright, got %v", sky)
	}
}

func TestSkyColorImported(t *testing.T) {
	*importedWorld = true
	defer func() { *importedWorld = false }()
	sky := SkyColor()
	if sky.X() > 0.2 || sky.Y() > 0.2 || sky.Z() > 0.2 {
		t.Errorf("imported sky should be dark, got %v", sky)
	}
}

func TestSetupSignalHandlerReceivesSIGINT(t *testing.T) {
	signal.Reset(syscall.SIGINT)
	ch := SetupSignalHandler()
	go func() {
		time.Sleep(50 * time.Millisecond)
		p, _ := os.FindProcess(os.Getpid())
		p.Signal(syscall.SIGINT)
	}()
	select {
	case sig := <-ch:
		if sig != syscall.SIGINT {
			t.Errorf("expected SIGINT, got %v", sig)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SIGINT")
	}
	signal.Reset(syscall.SIGINT, syscall.SIGTERM)
}

func TestIsTransparentClassification(t *testing.T) {
	if !IsTransparent(0) {
		t.Error("air (0) should be transparent")
	}
	if !IsTransparent(10) {
		t.Error("glass (10) should be transparent")
	}
	if !IsTransparent(15) {
		t.Error("leaves (15) should be transparent")
	}
	if !IsTransparent(17) {
		t.Error("plants (17) should be transparent")
	}
	if IsTransparent(1) {
		t.Error("grass (1) should not be transparent")
	}
	if IsTransparent(4) {
		t.Error("stone (4) should not be transparent")
	}
}
