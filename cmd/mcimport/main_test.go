package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/boltdb/bolt"
)

const testWorldDir = "../../Dungeon-of-Fears/Dungeon of Fears"
const testRegionDir = testWorldDir + "/region"

func hasTestData(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(testRegionDir, "r.0.0.mca")); err != nil {
		t.Skip("test data not available: Dungeon-of-Fears not found")
	}
}

func loadTestChunk(t *testing.T) map[string]interface{} {
	t.Helper()
	hasTestData(t)

	data, err := os.ReadFile(filepath.Join(testRegionDir, "r.0.0.mca"))
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 1024; i++ {
		offset := int(data[i*4])<<16 | int(data[i*4+1])<<8 | int(data[i*4+2])
		sectorCount := data[i*4+3]
		if offset == 0 && sectorCount == 0 {
			continue
		}

		byteOffset := offset * 4096
		length := int(binary.BigEndian.Uint32(data[byteOffset : byteOffset+4]))
		if data[byteOffset+4] != 2 {
			continue
		}

		reader, err := zlib.NewReader(bytes.NewReader(data[byteOffset+5 : byteOffset+4+length]))
		if err != nil {
			continue
		}
		nbtData, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			continue
		}

		r := bytes.NewReader(nbtData)
		chunk := parseNBTFull(r)
		if chunk != nil {
			return chunk
		}
	}
	t.Fatal("no valid chunks found in test region file")
	return nil
}

func TestNBTParsesChunkStructure(t *testing.T) {
	chunk := loadTestChunk(t)

	requiredKeys := []string{"xPos", "zPos", "sections", "Status"}
	for _, key := range requiredKeys {
		if _, ok := chunk[key]; !ok {
			t.Errorf("chunk missing required key %q", key)
		}
	}

	status, ok := chunk["Status"].(string)
	if !ok {
		t.Fatal("Status is not a string")
	}
	if status != "minecraft:full" {
		t.Errorf("expected Status=minecraft:full, got %q", status)
	}
}

func TestNBTSectionsHavePalettes(t *testing.T) {
	chunk := loadTestChunk(t)

	sections, ok := chunk["sections"].([]interface{})
	if !ok {
		t.Fatal("sections is not a list")
	}
	if len(sections) == 0 {
		t.Fatal("no sections in chunk")
	}

	paletteCount := 0
	for _, sec := range sections {
		secMap, ok := sec.(map[string]interface{})
		if !ok {
			continue
		}
		bs, ok := secMap["block_states"]
		if !ok {
			continue
		}
		bsMap, ok := bs.(map[string]interface{})
		if !ok {
			continue
		}
		palette, ok := bsMap["palette"]
		if !ok {
			continue
		}
		paletteList, ok := palette.([]interface{})
		if !ok {
			continue
		}
		if len(paletteList) > 0 {
			paletteCount++
			firstEntry, ok := paletteList[0].(map[string]interface{})
			if !ok {
				t.Error("palette entry is not a compound")
				continue
			}
			if _, ok := firstEntry["Name"]; !ok {
				t.Error("palette entry missing Name field")
			}
		}
	}
	if paletteCount == 0 {
		t.Error("no sections had block_states palettes")
	}
	t.Logf("found %d sections with palettes out of %d total sections", paletteCount, len(sections))
}

func TestUnpackBlockBasic(t *testing.T) {
	longs := []int64{0x0123456789ABCDEF}

	result := unpackBlock(longs, 0, 4)
	expected := 0xF
	if result != expected {
		t.Errorf("unpackBlock(0, 4) = %d, want %d", result, expected)
	}

	result = unpackBlock(longs, 1, 4)
	expected = 0xE
	if result != expected {
		t.Errorf("unpackBlock(1, 4) = %d, want %d", result, expected)
	}

	result = unpackBlock(longs, 15, 4)
	expected = 0x0
	if result != expected {
		t.Errorf("unpackBlock(15, 4) = %d, want %d", result, expected)
	}
}

func TestUnpackBlockOutOfBounds(t *testing.T) {
	longs := []int64{0xFF}
	result := unpackBlock(longs, 100, 4)
	if result != -1 {
		t.Errorf("out of bounds should return -1, got %d", result)
	}
}

func TestUnpackBlockVariousBitWidths(t *testing.T) {
	var packed uint64
	for i := 0; i < 16; i++ {
		packed |= uint64(i) << (i * 4)
	}
	longs := []int64{int64(packed)}

	for i := 0; i < 16; i++ {
		result := unpackBlock(longs, i, 4)
		if result != i {
			t.Errorf("unpackBlock(%d, 4) = %d, want %d", i, result, i)
		}
	}

	var packed5 uint64
	for i := 0; i < 12; i++ {
		packed5 |= uint64(i) << (i * 5)
	}
	longs5 := []int64{int64(packed5)}
	for i := 0; i < 12; i++ {
		result := unpackBlock(longs5, i, 5)
		if result != i {
			t.Errorf("unpackBlock(%d, 5) = %d, want %d", i, result, i)
		}
	}
}

func TestBlockMappingKnownTypes(t *testing.T) {
	unmapped := make(map[string]int)

	tests := []struct {
		mcBlock    string
		gcExpected int
	}{
		{"minecraft:air", 0},
		{"minecraft:grass_block", 1},
		{"minecraft:dirt", 1},
		{"minecraft:sand", 2},
		{"minecraft:stone", 4},
		{"minecraft:oak_log", 5},
		{"minecraft:cobblestone", 6},
		{"minecraft:stone_bricks", 7},
		{"minecraft:bricks", 8},
		{"minecraft:sandstone", 9},
		{"minecraft:glass", 10},
		{"minecraft:diamond_block", 11},
		{"minecraft:gold_block", 12},
		{"minecraft:iron_block", 13},
		{"minecraft:oak_leaves", 15},
		{"minecraft:spruce_leaves", 15},
	}

	for _, tt := range tests {
		result := mapBlock(tt.mcBlock, unmapped)
		if result != tt.gcExpected {
			t.Errorf("mapBlock(%q) = %d, want %d", tt.mcBlock, result, tt.gcExpected)
		}
	}
}

func TestBlockMappingSkippedTypes(t *testing.T) {
	unmapped := make(map[string]int)

	skipped := []string{
		"minecraft:stone_button",
		"minecraft:tripwire_hook",
		"minecraft:redstone_wire",
		"minecraft:wall_torch",
		"minecraft:rail",
	}
	for _, name := range skipped {
		result := mapBlock(name, unmapped)
		if result != 0 {
			t.Errorf("mapBlock(%q) = %d, want 0 (should be skipped)", name, result)
		}
	}
}

func TestBlockMappingFallbackHeuristic(t *testing.T) {
	unmapped := make(map[string]int)

	result := mapBlock("minecraft:unknown_stone_variant", unmapped)
	if result != 7 {
		t.Errorf("fallback for 'stone' keyword = %d, want 7", result)
	}

	result = mapBlock("minecraft:fancy_wood_log", unmapped)
	if result != 5 {
		t.Errorf("fallback for 'wood'+'log' keyword = %d, want 5", result)
	}

	result = mapBlock("minecraft:totally_unknown_block", unmapped)
	if result != 4 {
		t.Errorf("ultimate fallback = %d, want 4 (stone)", result)
	}

	if len(unmapped) != 3 {
		t.Errorf("expected 3 unmapped entries, got %d", len(unmapped))
	}
}

func TestPlantMapping(t *testing.T) {
	unmapped := make(map[string]int)

	plants := []struct {
		name     string
		expected int
	}{
		{"minecraft:short_grass", 17},
		{"minecraft:dandelion", 18},
		{"minecraft:poppy", 19},
		{"minecraft:cornflower", 20},
	}
	for _, tt := range plants {
		result := mapBlock(tt.name, unmapped)
		if result != tt.expected {
			t.Errorf("mapBlock(%q) = %d, want %d", tt.name, result, tt.expected)
		}
	}
}

func TestChunkIdMapping(t *testing.T) {
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
		{Vec3{15, 50, 15}, Vec3{0, 0, 0}},
		{Vec3{16, 50, 16}, Vec3{0, 0, 0}},
		{Vec3{47, 50, 47}, Vec3{1, 0, 1}},
	}

	for _, tt := range tests {
		result := tt.block.Chunkid()
		if result != tt.expected {
			t.Errorf("Vec3%v.Chunkid() = %v, want %v", tt.block, result, tt.expected)
		}
	}
}

func TestDBKeyRoundTrip(t *testing.T) {
	cid := Vec3{3, 0, -2}
	bid := Vec3{100, 42, -50}

	key := encodeBlockDbKey(cid, bid)
	gotCid, gotBid := decodeBlockDbKey(key)

	if gotCid != cid {
		t.Errorf("chunk id roundtrip: got %v, want %v", gotCid, cid)
	}
	if gotBid != bid {
		t.Errorf("block id roundtrip: got %v, want %v", gotBid, bid)
	}
}

func TestToIntConversions(t *testing.T) {
	if toInt(byte(255)) != -1 {
		t.Errorf("byte(255) should be -1 as int8, got %d", toInt(byte(255)))
	}
	if toInt(byte(0)) != 0 {
		t.Errorf("byte(0) should be 0, got %d", toInt(byte(0)))
	}
	if toInt(int32(42)) != 42 {
		t.Errorf("int32(42) should be 42, got %d", toInt(int32(42)))
	}
	if toInt(int32(-4)) != -4 {
		t.Errorf("int32(-4) should be -4, got %d", toInt(int32(-4)))
	}
}

func TestRegionFileBlockCounts(t *testing.T) {
	hasTestData(t)

	data, err := os.ReadFile(filepath.Join(testRegionDir, "r.0.0.mca"))
	if err != nil {
		t.Fatal(err)
	}

	chunkCount := 0
	totalSections := 0
	for i := 0; i < 1024; i++ {
		offset := int(data[i*4])<<16 | int(data[i*4+1])<<8 | int(data[i*4+2])
		sectorCount := data[i*4+3]
		if offset == 0 && sectorCount == 0 {
			continue
		}

		byteOffset := offset * 4096
		if byteOffset+5 > len(data) {
			continue
		}
		length := int(binary.BigEndian.Uint32(data[byteOffset : byteOffset+4]))
		if data[byteOffset+4] != 2 || byteOffset+4+length > len(data) {
			continue
		}

		reader, err := zlib.NewReader(bytes.NewReader(data[byteOffset+5 : byteOffset+4+length]))
		if err != nil {
			continue
		}
		nbtData, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			continue
		}

		r := bytes.NewReader(nbtData)
		chunk := parseNBTFull(r)
		if chunk == nil {
			continue
		}

		chunkCount++
		sections, ok := chunk["sections"].([]interface{})
		if ok {
			totalSections += len(sections)
		}
	}

	if chunkCount == 0 {
		t.Fatal("no chunks parsed from r.0.0.mca")
	}
	t.Logf("r.0.0.mca: %d chunks, %d total sections", chunkCount, totalSections)

	if chunkCount < 100 {
		t.Errorf("expected at least 100 chunks in region, got %d", chunkCount)
	}
}

func TestImportProducesBlocks(t *testing.T) {
	hasTestData(t)

	dbPath := t.TempDir() + "/test_import.db"
	os.Args = []string{"mcimport", testRegionDir, dbPath}

	os.Remove(dbPath)

	db, err := openTestDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	n, unmapped := importRegion(filepath.Join(testRegionDir, "r.0.0.mca"), db)
	db.Close()

	if n == 0 {
		t.Fatal("no blocks imported from r.0.0.mca")
	}
	t.Logf("imported %d blocks, %d unmapped types", n, len(unmapped))

	if n < 10000 {
		t.Errorf("expected at least 10000 blocks, got %d", n)
	}
}

func TestNetherRegionParsesChunks(t *testing.T) {
	netherDir := testWorldDir + "/DIM-1/region"
	regionFile := filepath.Join(netherDir, "r.0.0.mca")
	if _, err := os.Stat(regionFile); err != nil {
		t.Skip("nether region data not available")
	}

	data, err := os.ReadFile(regionFile)
	if err != nil {
		t.Fatal(err)
	}

	chunkCount := 0
	sectionCount := 0
	for i := 0; i < 1024; i++ {
		offset := int(data[i*4])<<16 | int(data[i*4+1])<<8 | int(data[i*4+2])
		sectorCount := data[i*4+3]
		if offset == 0 && sectorCount == 0 {
			continue
		}
		byteOffset := offset * 4096
		if byteOffset+5 > len(data) {
			continue
		}
		length := int(binary.BigEndian.Uint32(data[byteOffset : byteOffset+4]))
		if data[byteOffset+4] != 2 || byteOffset+4+length > len(data) {
			continue
		}
		reader, err := zlib.NewReader(bytes.NewReader(data[byteOffset+5 : byteOffset+4+length]))
		if err != nil {
			continue
		}
		nbtData, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			continue
		}
		r := bytes.NewReader(nbtData)
		chunk := parseNBTFull(r)
		if chunk == nil {
			continue
		}
		chunkCount++
		if sections, ok := chunk["sections"].([]interface{}); ok {
			sectionCount += len(sections)
		}
		if chunkCount >= 10 {
			break
		}
	}

	if chunkCount == 0 {
		t.Fatal("no nether chunks parsed")
	}
	t.Logf("nether: parsed %d chunks with %d sections", chunkCount, sectionCount)
	if sectionCount == 0 {
		t.Error("no sections found in nether chunks")
	}
}

func TestNetherBlockTypes(t *testing.T) {
	netherDir := testWorldDir + "/DIM-1/region"
	if _, err := os.Stat(filepath.Join(netherDir, "r.0.0.mca")); err != nil {
		t.Skip("nether region data not available")
	}

	unmapped := make(map[string]int)
	netherBlocks := []string{
		"minecraft:netherrack",
		"minecraft:soul_sand",
		"minecraft:basalt",
		"minecraft:blackstone",
		"minecraft:nether_bricks",
	}
	for _, name := range netherBlocks {
		result := mapBlock(name, unmapped)
		if result == 0 {
			t.Errorf("nether block %q should map to non-zero type, got 0", name)
		}
	}
}

func TestParseLevelDatSpawn(t *testing.T) {
	hasTestData(t)

	x, y, z, err := ParseLevelDat(filepath.Join(testWorldDir, "level.dat"))
	if err != nil {
		t.Fatal(err)
	}

	if x != -111 || y != 13 || z != -5 {
		t.Errorf("spawn = %d,%d,%d, want -111,13,-5", x, y, z)
	}
	t.Logf("level.dat spawn: %d, %d, %d", x, y, z)
}

func TestParseLevelDatName(t *testing.T) {
	hasTestData(t)

	name, err := ParseLevelDatName(filepath.Join(testWorldDir, "level.dat"))
	if err != nil {
		t.Fatal(err)
	}

	if name == "" {
		t.Error("level name should not be empty")
	}
	t.Logf("level name: %q", name)
}

func TestParseLevelDatMissing(t *testing.T) {
	_, _, _, err := ParseLevelDat("/nonexistent/level.dat")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func openTestDB(path string) (*bolt.DB, error) {
	db, err := bolt.Open(path, 0666, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte("block")); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte("chunk")); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte("camera")); err != nil {
			return err
		}
		return nil
	})
	return db, err
}
