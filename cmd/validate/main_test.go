package main

import (
	"os"
	"testing"
)

const testDB = "../../dungeon.db"

func hasTestDB(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(testDB); err != nil {
		t.Skip("test db not available")
	}
}

func TestValidatorLoadsWorld(t *testing.T) {
	hasTestDB(t)

	v, err := NewWorldValidator(testDB)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()

	if len(v.blocks) == 0 {
		t.Fatal("no blocks loaded")
	}
	t.Logf("loaded %d blocks in %d chunks", len(v.blocks), len(v.chunks))
}

func TestValidatorBlockCounts(t *testing.T) {
	hasTestDB(t)

	v, err := NewWorldValidator(testDB)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()

	r := v.Validate()

	if r.TotalBlocks < 100000 {
		t.Errorf("expected at least 100000 blocks, got %d", r.TotalBlocks)
	}
	if r.TotalChunks < 10 {
		t.Errorf("expected at least 10 chunks, got %d", r.TotalChunks)
	}
	if len(r.Errors) > 0 {
		t.Errorf("validation errors: %v", r.Errors)
	}
}

func TestValidatorBlockTypesValid(t *testing.T) {
	hasTestDB(t)

	v, err := NewWorldValidator(testDB)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()

	for pos, tp := range v.blocks {
		if tp < 0 || tp > 64 {
			t.Errorf("invalid block type %d at %v", tp, pos)
		}
	}
}

func TestValidatorWorldBounds(t *testing.T) {
	hasTestDB(t)

	v, err := NewWorldValidator(testDB)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()

	r := v.Validate()

	if r.MinY < 0 {
		t.Errorf("MinY should be >= 0 (we skip negative Y), got %d", r.MinY)
	}
	if r.MaxY > 400 {
		t.Errorf("MaxY seems too high: %d", r.MaxY)
	}
	worldWidth := r.MaxX - r.MinX
	worldDepth := r.MaxZ - r.MinZ
	t.Logf("world size: %dx%d blocks (X x Z), height %d-%d", worldWidth, worldDepth, r.MinY, r.MaxY)
}

func TestValidatorGetBlock(t *testing.T) {
	hasTestDB(t)

	v, err := NewWorldValidator(testDB)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()

	solidFound := 0
	airFound := 0
	for y := 0; y < 100; y++ {
		tp := v.GetBlock(Vec3{0, y, 0})
		if tp != 0 {
			solidFound++
		} else {
			airFound++
		}
	}
	t.Logf("column at (0,0): %d solid, %d air in Y 0-99", solidFound, airFound)
	if solidFound == 0 {
		t.Error("expected at least some solid blocks at origin column")
	}
}

func TestValidatorSpawnHasGround(t *testing.T) {
	hasTestDB(t)

	v, err := NewWorldValidator(testDB)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()

	r := v.Validate()

	for _, e := range r.Errors {
		if e == "spawn at (0.0, 52.0, 0.0) has no solid ground below" {
			t.Error("spawn point has no solid ground")
		}
	}
}
