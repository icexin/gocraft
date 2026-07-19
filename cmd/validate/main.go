package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"

	"github.com/boltdb/bolt"
)

type Vec3 struct {
	X, Y, Z int
}

func (v Vec3) Chunkid() Vec3 {
	return Vec3{
		int(math.Floor(float64(v.X) / 32)),
		0,
		int(math.Floor(float64(v.Z) / 32)),
	}
}

type WorldValidator struct {
	db     *bolt.DB
	blocks map[Vec3]int
	chunks map[Vec3]int
}

func NewWorldValidator(dbPath string) (*WorldValidator, error) {
	db, err := bolt.Open(dbPath, 0444, &bolt.Options{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	v := &WorldValidator{
		db:     db,
		blocks: make(map[Vec3]int),
		chunks: make(map[Vec3]int),
	}

	err = db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte("block"))
		if bkt == nil {
			return fmt.Errorf("no block bucket")
		}
		return bkt.ForEach(func(k, val []byte) error {
			_, bid := decodeBlockDbKey(k)
			w := int(binary.LittleEndian.Uint32(val))
			if w != 0 {
				v.blocks[bid] = w
				cid := bid.Chunkid()
				v.chunks[cid]++
			}
			return nil
		})
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return v, nil
}

func (v *WorldValidator) Close() {
	v.db.Close()
}

func (v *WorldValidator) GetBlock(id Vec3) int {
	if tp, ok := v.blocks[id]; ok {
		return tp
	}
	return 0
}

type ValidationResult struct {
	TotalBlocks    int
	TotalChunks    int
	BlockTypeCounts map[int]int
	MinY, MaxY     int
	MinX, MaxX     int
	MinZ, MaxZ     int
	Errors         []string
	Warnings       []string
}

func (v *WorldValidator) Validate() *ValidationResult {
	r := &ValidationResult{
		BlockTypeCounts: make(map[int]int),
		MinY:           math.MaxInt32,
		MaxY:           math.MinInt32,
		MinX:           math.MaxInt32,
		MaxX:           math.MinInt32,
		MinZ:           math.MaxInt32,
		MaxZ:           math.MinInt32,
	}

	r.TotalBlocks = len(v.blocks)
	r.TotalChunks = len(v.chunks)

	for pos, tp := range v.blocks {
		r.BlockTypeCounts[tp]++

		if pos.Y < r.MinY {
			r.MinY = pos.Y
		}
		if pos.Y > r.MaxY {
			r.MaxY = pos.Y
		}
		if pos.X < r.MinX {
			r.MinX = pos.X
		}
		if pos.X > r.MaxX {
			r.MaxX = pos.X
		}
		if pos.Z < r.MinZ {
			r.MinZ = pos.Z
		}
		if pos.Z > r.MaxZ {
			r.MaxZ = pos.Z
		}

		if tp < 0 || tp > 64 {
			r.Errors = append(r.Errors, fmt.Sprintf("invalid block type %d at %v", tp, pos))
		}
	}

	if r.TotalBlocks == 0 {
		r.Errors = append(r.Errors, "no blocks in world")
	}

	emptyChunks := 0
	for _, count := range v.chunks {
		if count == 0 {
			emptyChunks++
		}
	}
	if emptyChunks > 0 {
		r.Warnings = append(r.Warnings, fmt.Sprintf("%d empty chunks", emptyChunks))
	}

	v.validateCollision(r)
	v.validateSpawn(r)

	return r
}

func (v *WorldValidator) validateCollision(r *ValidationResult) {
	testPoints := []struct {
		name string
		pos  Vec3
	}{
		{"origin", Vec3{0, 0, 0}},
		{"center", Vec3{(r.MinX + r.MaxX) / 2, r.MaxY, (r.MinZ + r.MaxZ) / 2}},
	}

	for _, tp := range testPoints {
		maxY := -1
		for y := r.MaxY; y >= r.MinY; y-- {
			block := v.GetBlock(Vec3{tp.pos.X, y, tp.pos.Z})
			if block != 0 && isObstacle(block) {
				maxY = y
				break
			}
		}
		if maxY >= 0 {
			above := v.GetBlock(Vec3{tp.pos.X, maxY + 1, tp.pos.Z})
			if above != 0 && isObstacle(above) {
				r.Warnings = append(r.Warnings, fmt.Sprintf("no headroom at %s (%d, %d, %d)", tp.name, tp.pos.X, maxY+1, tp.pos.Z))
			}
		}
	}

	solidColumns := 0
	hollowColumns := 0
	sampleStep := 16
	for x := r.MinX; x <= r.MaxX; x += sampleStep {
		for z := r.MinZ; z <= r.MaxZ; z += sampleStep {
			hasSolid := false
			for y := r.MinY; y <= r.MaxY; y++ {
				if v.GetBlock(Vec3{x, y, z}) != 0 {
					hasSolid = true
					break
				}
			}
			if hasSolid {
				solidColumns++
			} else {
				hollowColumns++
			}
		}
	}
	r.Warnings = append(r.Warnings, fmt.Sprintf("column sample: %d solid, %d hollow (step=%d)", solidColumns, hollowColumns, sampleStep))

	dropTests := 0
	dropSuccesses := 0
	for x := r.MinX; x <= r.MaxX; x += sampleStep * 2 {
		for z := r.MinZ; z <= r.MaxZ; z += sampleStep * 2 {
			topY := -1
			for y := r.MaxY; y >= r.MinY; y-- {
				if v.GetBlock(Vec3{x, y, z}) != 0 && isObstacle(v.GetBlock(Vec3{x, y, z})) {
					topY = y
					break
				}
			}
			if topY < 0 {
				continue
			}
			dropTests++
			standY := topY + 1
			below := v.GetBlock(Vec3{x, standY - 1, z})
			if isObstacle(below) {
				dropSuccesses++
			}
		}
	}
	if dropTests > 0 {
		pct := float64(dropSuccesses) / float64(dropTests) * 100
		if pct < 80 {
			r.Errors = append(r.Errors, fmt.Sprintf("only %.0f%% of drop tests found solid ground (%d/%d)", pct, dropSuccesses, dropTests))
		}
	}
}

func (v *WorldValidator) validateSpawn(r *ValidationResult) {
	var spawnY float32
	v.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte("camera"))
		if bkt == nil {
			return nil
		}
		val := bkt.Get([]byte("camera"))
		if val == nil {
			r.Warnings = append(r.Warnings, "no saved camera state")
			return nil
		}
		var state struct {
			X, Y, Z float32
			Rx, Ry  float32
		}
		buf := bytes.NewBuffer(val)
		binary.Read(buf, binary.LittleEndian, &state)
		spawnY = state.Y

		hasSolidBelow := false
		for y := int(state.Y); y >= r.MinY; y-- {
			if v.GetBlock(Vec3{int(math.Round(float64(state.X))), y, int(math.Round(float64(state.Z)))}) != 0 {
				hasSolidBelow = true
				break
			}
		}
		if !hasSolidBelow {
			r.Errors = append(r.Errors, fmt.Sprintf("spawn at (%.1f, %.1f, %.1f) has no solid ground below", state.X, state.Y, state.Z))
		}
		return nil
	})
	_ = spawnY
}

func isObstacle(tp int) bool {
	if tp >= 17 && tp <= 31 {
		return false
	}
	switch tp {
	case -1:
		return true
	case 0:
		return false
	default:
		return true
	}
}

func decodeBlockDbKey(b []byte) (Vec3, Vec3) {
	if len(b) != 4*5 {
		log.Panicf("bad db key length:%d", len(b))
	}
	buf := bytes.NewBuffer(b)
	var arr [5]int32
	binary.Read(buf, binary.LittleEndian, &arr)
	cid := Vec3{int(arr[0]), 0, int(arr[1])}
	bid := Vec3{int(arr[2]), int(arr[3]), int(arr[4])}
	return cid, bid
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: validate <world.db>")
	}

	v, err := NewWorldValidator(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer v.Close()

	r := v.Validate()

	fmt.Println("=== World Validation Report ===")
	fmt.Printf("Total blocks: %d\n", r.TotalBlocks)
	fmt.Printf("Total chunks: %d\n", r.TotalChunks)
	fmt.Printf("World bounds: X[%d, %d] Y[%d, %d] Z[%d, %d]\n",
		r.MinX, r.MaxX, r.MinY, r.MaxY, r.MinZ, r.MaxZ)

	fmt.Println("\nBlock type distribution:")
	blockNames := map[int]string{
		1: "grass/dirt", 2: "sand", 3: "gravel/clay", 4: "stone",
		5: "wood/planks", 6: "cobblestone", 7: "stone_bricks", 8: "bricks",
		9: "sandstone", 10: "glass", 11: "diamond", 12: "gold",
		13: "iron", 14: "redstone", 15: "leaves", 16: "cloud",
	}
	for tp, count := range r.BlockTypeCounts {
		name := blockNames[tp]
		if name == "" {
			if tp >= 17 && tp <= 31 {
				name = "plant"
			} else {
				name = fmt.Sprintf("type_%d", tp)
			}
		}
		fmt.Printf("  %s (%d): %d blocks\n", name, tp, count)
	}

	if len(r.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range r.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}

	if len(r.Errors) > 0 {
		fmt.Println("\nERRORS:")
		for _, e := range r.Errors {
			fmt.Printf("  !! %s\n", e)
		}
		os.Exit(1)
	}

	fmt.Println("\n=== VALIDATION PASSED ===")
}
