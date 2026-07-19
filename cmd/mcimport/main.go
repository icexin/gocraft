package main

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"math/bits"
	"os"
	"path/filepath"
	"strings"

	"github.com/boltdb/bolt"
)

var blockMapping = map[string]int{
	"minecraft:grass_block":              1,
	"minecraft:dirt":                     1,
	"minecraft:dirt_path":                1,
	"minecraft:rooted_dirt":              1,
	"minecraft:coarse_dirt":              1,
	"minecraft:podzol":                   1,
	"minecraft:mud":                      1,
	"minecraft:muddy_mangrove_roots":     1,
	"minecraft:mangrove_roots":           5,
	"minecraft:sand":                     2,
	"minecraft:red_sand":                 2,
	"minecraft:soul_sand":                2,
	"minecraft:soul_soil":                2,
	"minecraft:gravel":                   3,
	"minecraft:stone":                    4,
	"minecraft:andesite":                 4,
	"minecraft:diorite":                  4,
	"minecraft:granite":                  4,
	"minecraft:tuff":                     4,
	"minecraft:deepslate":                4,
	"minecraft:cobbled_deepslate":        4,
	"minecraft:polished_deepslate":       4,
	"minecraft:deepslate_bricks":         4,
	"minecraft:cracked_deepslate_bricks": 4,
	"minecraft:chiseled_deepslate":       4,
	"minecraft:smooth_basalt":            4,
	"minecraft:basalt":                   4,
	"minecraft:polished_basalt":          4,
	"minecraft:polished_andesite":        4,
	"minecraft:polished_granite":         4,
	"minecraft:smooth_stone":             4,
	"minecraft:bedrock":                  4,
	"minecraft:obsidian":                 4,
	"minecraft:netherrack":               4,
	"minecraft:blackstone":               4,
	"minecraft:polished_blackstone":      4,
	"minecraft:oak_log":                  5,
	"minecraft:spruce_log":               5,
	"minecraft:birch_log":                5,
	"minecraft:jungle_log":               5,
	"minecraft:acacia_log":               5,
	"minecraft:dark_oak_log":             5,
	"minecraft:crimson_stem":             5,
	"minecraft:warped_stem":              5,
	"minecraft:stripped_spruce_log":       5,
	"minecraft:stripped_oak_log":          5,
	"minecraft:stripped_dark_oak_log":     5,
	"minecraft:stripped_acacia_wood":      5,
	"minecraft:bamboo_block":             5,
	"minecraft:cobblestone":              6,
	"minecraft:mossy_cobblestone":         6,
	"minecraft:stone_bricks":             7,
	"minecraft:mossy_stone_bricks":       7,
	"minecraft:cracked_stone_bricks":     7,
	"minecraft:chiseled_stone_bricks":    7,
	"minecraft:bricks":                   8,
	"minecraft:mud_bricks":               8,
	"minecraft:nether_bricks":            8,
	"minecraft:red_nether_bricks":        8,
	"minecraft:chiseled_nether_bricks":   8,
	"minecraft:polished_blackstone_bricks": 8,
	"minecraft:purpur_pillar":            8,
	"minecraft:sandstone":                9,
	"minecraft:cut_sandstone":            9,
	"minecraft:chiseled_sandstone":       9,
	"minecraft:red_sandstone":            9,
	"minecraft:cut_red_sandstone":        9,
	"minecraft:chiseled_red_sandstone":   9,
	"minecraft:glass":                    10,
	"minecraft:glass_pane":               10,
	"minecraft:blue_stained_glass":       10,
	"minecraft:red_stained_glass":        10,
	"minecraft:magenta_stained_glass":    10,
	"minecraft:diamond_block":            11,
	"minecraft:gold_block":               12,
	"minecraft:iron_block":               13,
	"minecraft:redstone_block":           14,
	"minecraft:oak_leaves":               15,
	"minecraft:spruce_leaves":            15,
	"minecraft:birch_leaves":             15,
	"minecraft:jungle_leaves":            15,
	"minecraft:acacia_leaves":            15,
	"minecraft:dark_oak_leaves":          15,
	"minecraft:azalea_leaves":            15,
	"minecraft:flowering_azalea_leaves":  15,
	"minecraft:nether_wart_block":        15,
	"minecraft:oak_planks":               5,
	"minecraft:spruce_planks":            5,
	"minecraft:birch_planks":             5,
	"minecraft:jungle_planks":            5,
	"minecraft:acacia_planks":            5,
	"minecraft:dark_oak_planks":          5,
	"minecraft:crimson_planks":           5,
	"minecraft:warped_planks":            5,
	"minecraft:bamboo_mosaic":            5,
	"minecraft:bookshelf":                5,
	"minecraft:chiseled_bookshelf":       5,
	"minecraft:crafting_table":           5,
	"minecraft:oak_slab":                 5,
	"minecraft:spruce_slab":              5,
	"minecraft:birch_slab":               5,
	"minecraft:jungle_slab":              5,
	"minecraft:dark_oak_slab":            5,
	"minecraft:cobblestone_slab":         6,
	"minecraft:stone_brick_slab":         7,
	"minecraft:smooth_stone_slab":        4,
	"minecraft:brick_slab":               8,
	"minecraft:nether_brick_slab":        8,
	"minecraft:red_nether_brick_slab":    8,
	"minecraft:sandstone_slab":           9,
	"minecraft:red_sandstone_slab":       9,
	"minecraft:cut_sandstone_slab":       9,
	"minecraft:polished_andesite_slab":   4,
	"minecraft:diorite_slab":             4,
	"minecraft:granite_slab":             4,
	"minecraft:andesite_slab":            4,
	"minecraft:cobbled_deepslate_slab":   4,
	"minecraft:polished_deepslate_slab":  4,
	"minecraft:deepslate_brick_stairs":   4,
	"minecraft:mossy_cobblestone_slab":   6,
	"minecraft:mossy_stone_brick_slab":   7,
	"minecraft:purpur_slab":              8,
	"minecraft:waxed_cut_copper_slab":    13,
	"minecraft:oak_stairs":               5,
	"minecraft:birch_stairs":             5,
	"minecraft:dark_oak_stairs":          5,
	"minecraft:jungle_stairs":            5,
	"minecraft:bamboo_stairs":            5,
	"minecraft:cobblestone_stairs":       6,
	"minecraft:stone_brick_stairs":       7,
	"minecraft:brick_stairs":             8,
	"minecraft:nether_brick_stairs":      8,
	"minecraft:red_nether_brick_stairs":  8,
	"minecraft:red_sandstone_stairs":     9,
	"minecraft:polished_andesite_stairs": 4,
	"minecraft:polished_deepslate_stairs": 4,
	"minecraft:polished_granite_stairs":  4,
	"minecraft:diorite_stairs":           4,
	"minecraft:purpur_stairs":            8,
	"minecraft:hay_block":                9,
	"minecraft:dried_kelp_block":         3,
	"minecraft:black_terracotta":         3,
	"minecraft:yellow_glazed_terracotta": 3,
	"minecraft:light_gray_glazed_terracotta": 3,
	"minecraft:orange_glazed_terracotta": 3,
	"minecraft:red_glazed_terracotta":    3,
	"minecraft:pink_glazed_terracotta":   3,
	"minecraft:white_glazed_terracotta":  3,
	"minecraft:purple_glazed_terracotta": 3,
	"minecraft:red_concrete_powder":      3,
	"minecraft:green_concrete_powder":    3,
	"minecraft:magenta_concrete":         3,
	"minecraft:black_concrete":           3,
	"minecraft:pearlescent_froglight":    10,
	"minecraft:ochre_froglight":          10,
	"minecraft:sculk":                    4,
	"minecraft:packed_ice":               10,
	"minecraft:blue_ice":                 10,
	"minecraft:slime_block":              10,
	"minecraft:waxed_weathered_copper":   13,
	"minecraft:waxed_copper_block":       13,
	"minecraft:target":                   3,
	"minecraft:barrel":                   5,
	"minecraft:lectern":                  5,
	"minecraft:jukebox":                  5,
	"minecraft:beehive":                  5,
	"minecraft:furnace":                  6,
	"minecraft:dispenser":                6,
	"minecraft:observer":                 6,
	"minecraft:piston":                   6,
	"minecraft:sticky_piston":            6,
	"minecraft:spawner":                  13,
	"minecraft:enchanting_table":         4,
	"minecraft:anvil":                    13,
	"minecraft:chipped_anvil":            13,
	"minecraft:damaged_anvil":            13,
	"minecraft:cauldron":                 13,
	"minecraft:water_cauldron":           13,
	"minecraft:iron_door":                13,
	"minecraft:oak_door":                 5,
	"minecraft:spruce_door":              5,
	"minecraft:birch_door":               5,
	"minecraft:jungle_door":              5,
	"minecraft:dark_oak_door":            5,
	"minecraft:cherry_door":              5,
	"minecraft:acacia_door":              5,
	"minecraft:iron_bars":                13,
	"minecraft:iron_trapdoor":            13,
	"minecraft:chain":                    13,
	"minecraft:oak_fence":                5,
	"minecraft:spruce_fence":             5,
	"minecraft:dark_oak_fence":           5,
	"minecraft:jungle_fence":             5,
	"minecraft:cobblestone_wall":         6,
	"minecraft:stone_brick_wall":         7,
	"minecraft:granite_wall":             4,
	"minecraft:nether_brick_wall":        8,
	"minecraft:polished_deepslate_wall":  4,
	"minecraft:polished_blackstone_wall": 4,
	"minecraft:chest":                    5,
	"minecraft:trapped_chest":            5,
	"minecraft:composter":                5,
	"minecraft:command_block":            4,
	"minecraft:repeating_command_block":  4,
	"minecraft:barrier":                  0,
	"minecraft:powder_snow":              0,
	"minecraft:light":                    0,
	"minecraft:water":                    10,
	"minecraft:lava":                     14,
	"minecraft:brown_wool":               3,
	"minecraft:red_wool":                 3,
	"minecraft:brown_carpet":             0,
	"minecraft:red_carpet":               0,
	"minecraft:orange_carpet":            0,
	"minecraft:scaffolding":              0,
	"minecraft:ladder":                   0,
	"minecraft:pointed_dripstone":        4,
	"minecraft:cobweb":                   0,
}

var plantMapping = map[string]int{
	"minecraft:short_grass":        17,
	"minecraft:tall_grass":         17,
	"minecraft:fern":               17,
	"minecraft:large_fern":         17,
	"minecraft:dandelion":          18,
	"minecraft:poppy":              19,
	"minecraft:cornflower":         20,
	"minecraft:azure_bluet":        21,
	"minecraft:oxeye_daisy":        22,
	"minecraft:wither_rose":        23,
	"minecraft:crimson_fungus":     23,
	"minecraft:crimson_roots":      23,
	"minecraft:dead_bush":          17,
	"minecraft:brown_mushroom":     23,
	"minecraft:moss_carpet":        17,
	"minecraft:flowering_azalea":   17,
	"minecraft:azalea":             17,
	"minecraft:spruce_sapling":     17,
	"minecraft:seagrass":           17,
	"minecraft:tall_seagrass":      17,
	"minecraft:hanging_roots":      17,
	"minecraft:glow_lichen":        17,
	"minecraft:vine":               17,
	"minecraft:fire":               0,
	"minecraft:warped_nylium":      1,
	"minecraft:crimson_nylium":     1,
}

var skipBlocks = map[string]bool{
	"minecraft:air":                            true,
	"minecraft:cave_air":                       true,
	"minecraft:void_air":                       true,
	"minecraft:stone_button":                   true,
	"minecraft:crimson_button":                 true,
	"minecraft:polished_blackstone_button":     true,
	"minecraft:dark_oak_button":                true,
	"minecraft:birch_button":                   true,
	"minecraft:tripwire_hook":                  true,
	"minecraft:redstone_wire":                  true,
	"minecraft:repeater":                       true,
	"minecraft:lever":                          true,
	"minecraft:rail":                           true,
	"minecraft:torch":                          true,
	"minecraft:wall_torch":                     true,
	"minecraft:soul_wall_torch":                true,
	"minecraft:soul_lantern":                   true,
	"minecraft:lantern":                        true,
	"minecraft:light_blue_candle":              true,
	"minecraft:pink_candle":                    true,
	"minecraft:black_candle":                   true,
	"minecraft:brown_candle":                   true,
	"minecraft:purple_candle":                  true,
	"minecraft:candle":                         true,
	"minecraft:yellow_candle":                  true,
	"minecraft:blue_candle":                    true,
	"minecraft:light_gray_candle":              true,
	"minecraft:lightning_rod":                   true,
	"minecraft:crimson_pressure_plate":         true,
	"minecraft:birch_pressure_plate":           true,
	"minecraft:warped_pressure_plate":          true,
	"minecraft:spruce_pressure_plate":          true,
	"minecraft:stone_pressure_plate":           true,
	"minecraft:heavy_weighted_pressure_plate":  true,
	"minecraft:polished_blackstone_pressure_plate": true,
	"minecraft:decorated_pot":                  true,
	"minecraft:flower_pot":                     true,
	"minecraft:potted_dead_bush":               true,
	"minecraft:magenta_bed":                    true,
	"minecraft:orange_bed":                     true,
	"minecraft:light_blue_bed":                 true,
	"minecraft:blue_bed":                       true,
	"minecraft:yellow_bed":                     true,
	"minecraft:gray_bed":                       true,
	"minecraft:pink_bed":                       true,
	"minecraft:light_gray_bed":                 true,
	"minecraft:brown_bed":                      true,
	"minecraft:cake":                           true,
	"minecraft:campfire":                       true,
	"minecraft:turtle_egg":                     true,
	"minecraft:sea_pickle":                     true,
	"minecraft:nether_portal":                  true,
	"minecraft:end_portal":                     true,
	"minecraft:creeper_head":                   true,
	"minecraft:dragon_wall_head":               true,
	"minecraft:spruce_trapdoor":                true,
	"minecraft:jungle_trapdoor":                true,
	"minecraft:spruce_door":                    true,
	"minecraft:cherry_sign":                    true,
	"minecraft:crimson_sign":                   true,
	"minecraft:jungle_wall_sign":               true,
	"minecraft:cherry_wall_sign":               true,
	"minecraft:dark_oak_wall_sign":             true,
	"minecraft:birch_wall_sign":                true,
	"minecraft:crimson_wall_sign":              true,
	"minecraft:bamboo_wall_sign":               true,
	"minecraft:acacia_wall_sign":               true,
	"minecraft:warped_wall_sign":               true,
}

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

const (
	tagEnd       = 0
	tagByte      = 1
	tagShort     = 2
	tagInt       = 3
	tagLong      = 4
	tagFloat     = 5
	tagDouble    = 6
	tagByteArray = 7
	tagString    = 8
	tagList      = 9
	tagCompound  = 10
	tagIntArray  = 11
	tagLongArray = 12
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal("usage: mcimport <world_dir> <output.db>")
	}

	worldDir := os.Args[1]
	dbPath := os.Args[2]
	regionDir := filepath.Join(worldDir, "region")

	os.Remove(dbPath)
	db, err := bolt.Open(dbPath, 0666, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

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
	if err != nil {
		log.Fatal(err)
	}

	totalBlocks := 0
	unmapped := make(map[string]int)

	regionDirs := []struct {
		path string
		name string
	}{
		{regionDir, "overworld"},
		{filepath.Join(worldDir, "DIM-1", "region"), "nether"},
		{filepath.Join(worldDir, "DIM1", "region"), "the_end"},
	}

	for _, rd := range regionDirs {
		files, err := filepath.Glob(filepath.Join(rd.path, "r.*.mca"))
		if err != nil || len(files) == 0 {
			continue
		}
		fmt.Printf("\nimporting %s (%d region files)...\n", rd.name, len(files))
		for _, f := range files {
			n, um := importRegion(f, db)
			totalBlocks += n
			for k, v := range um {
				unmapped[k] += v
			}
		}
	}

	if totalBlocks == 0 {
		log.Fatal("no blocks imported from any dimension")
	}

	if len(unmapped) > 0 {
		fmt.Println("\nunmapped block types:")
		for k, v := range unmapped {
			fmt.Printf("  %s: %d\n", k, v)
		}
	}

	spawnX, spawnY, spawnZ := 0, 0, 0
	levelDatPath := filepath.Join(worldDir, "level.dat")
	if sx, sy, sz, err := ParseLevelDat(levelDatPath); err == nil {
		spawnX, spawnY, spawnZ = sx, sy+2, sz
		fmt.Printf("spawn from level.dat: %d, %d, %d\n", spawnX, spawnY, spawnZ)
	} else {
		log.Printf("level.dat not available (%v), using heuristic", err)
		spawnY = findSpawnY(db) + 2
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte("camera"))
		state := struct {
			X, Y, Z float32
			Rx, Ry  float32
		}{float32(spawnX), float32(spawnY), float32(spawnZ), -90, 0}
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.LittleEndian, &state)
		return bkt.Put([]byte("camera"), buf.Bytes())
	})
	if err != nil {
		log.Printf("warning: failed to set spawn: %v", err)
	}

	fmt.Printf("\nimported %d blocks, spawn at %d, %d, %d\n", totalBlocks, spawnX, spawnY, spawnZ)
	fmt.Printf("output: %s\n", dbPath)
}

func findSpawnY(db *bolt.DB) int {
	maxY := 0
	db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte("block"))
		c := bkt.Cursor()
		target := encodeBlockDbKey(Vec3{0, 0, 0}, Vec3{0, 0, 0})
		for k, v := c.Seek(target); k != nil; k, v = c.Next() {
			cid, bid := decodeBlockDbKey(k)
			if cid.X != 0 || cid.Z != 0 {
				break
			}
			w := int(binary.LittleEndian.Uint32(v))
			if w != 0 && bid.X == 0 && bid.Z == 0 && bid.Y > maxY {
				maxY = bid.Y
			}
		}
		return nil
	})
	if maxY == 0 {
		maxY = 64
	}
	return maxY
}

func importRegion(path string, db *bolt.DB) (int, map[string]int) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 8192 {
		return 0, nil
	}

	fname := filepath.Base(path)
	fmt.Printf("processing %s...\n", fname)

	totalBlocks := 0
	unmapped := make(map[string]int)
	batch := make(map[Vec3]int)

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

		xPosRaw, _ := chunk["xPos"]
		zPosRaw, _ := chunk["zPos"]
		mcChunkX := toInt(xPosRaw)
		mcChunkZ := toInt(zPosRaw)

		sections, ok := chunk["sections"]
		if !ok {
			continue
		}
		sectionList, ok := sections.([]interface{})
		if !ok {
			continue
		}

		for _, sec := range sectionList {
			secMap, ok := sec.(map[string]interface{})
			if !ok {
				continue
			}

			sectionY := toInt(secMap["Y"])
			worldYBase := sectionY * 16

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

			blockNames := make([]string, len(paletteList))
			for pi, p := range paletteList {
				pMap, ok := p.(map[string]interface{})
				if !ok {
					continue
				}
				if name, ok := pMap["Name"]; ok {
					blockNames[pi] = name.(string)
				}
			}

			if len(paletteList) == 1 {
				gcType := mapBlock(blockNames[0], unmapped)
				if gcType <= 0 {
					continue
				}
				for dy := 0; dy < 16; dy++ {
					for dz := 0; dz < 16; dz++ {
						for dx := 0; dx < 16; dx++ {
							worldX := mcChunkX*16 + dx
							worldY := worldYBase + dy
							worldZ := mcChunkZ*16 + dz
							if worldY < 0 {
								continue
							}
							batch[Vec3{worldX, worldY, worldZ}] = gcType
							totalBlocks++
						}
					}
				}
			} else {
				longData, ok := bsMap["data"]
				if !ok {
					continue
				}
				longs, ok := longData.([]int64)
				if !ok {
					continue
				}

				bitsPerBlock := max(4, bits.Len(uint(len(paletteList)-1)))

				for dy := 0; dy < 16; dy++ {
					for dz := 0; dz < 16; dz++ {
						for dx := 0; dx < 16; dx++ {
							blockIdx := dy*16*16 + dz*16 + dx
							paletteIdx := unpackBlock(longs, blockIdx, bitsPerBlock)
							if paletteIdx < 0 || paletteIdx >= len(blockNames) {
								continue
							}

							gcType := mapBlock(blockNames[paletteIdx], unmapped)
							if gcType <= 0 {
								continue
							}

							worldX := mcChunkX*16 + dx
							worldY := worldYBase + dy
							worldZ := mcChunkZ*16 + dz
							if worldY < 0 {
								continue
							}

							batch[Vec3{worldX, worldY, worldZ}] = gcType
							totalBlocks++
						}
					}
				}
			}

			if len(batch) >= 50000 {
				flushBatch(db, batch)
				batch = make(map[Vec3]int)
			}
		}
	}

	if len(batch) > 0 {
		flushBatch(db, batch)
	}

	fmt.Printf("  %s: %d blocks imported\n", fname, totalBlocks)
	return totalBlocks, unmapped
}

func mapBlock(name string, unmapped map[string]int) int {
	if skipBlocks[name] {
		return 0
	}
	if strings.HasSuffix(name, "_air") || name == "minecraft:air" {
		return 0
	}
	if t, ok := blockMapping[name]; ok {
		return t
	}
	if t, ok := plantMapping[name]; ok {
		return t
	}
	unmapped[name]++
	if strings.Contains(name, "stone") || strings.Contains(name, "brick") {
		return 7
	}
	if strings.Contains(name, "wood") || strings.Contains(name, "log") || strings.Contains(name, "plank") {
		return 5
	}
	if strings.Contains(name, "sand") {
		return 2
	}
	if strings.Contains(name, "dirt") || strings.Contains(name, "grass") {
		return 1
	}
	if strings.Contains(name, "leaves") {
		return 15
	}
	if strings.Contains(name, "iron") || strings.Contains(name, "metal") {
		return 13
	}
	if strings.Contains(name, "glass") {
		return 10
	}
	return 4
}

func flushBatch(db *bolt.DB, batch map[Vec3]int) {
	err := db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte("block"))
		for pos, w := range batch {
			cid := pos.Chunkid()
			key := encodeBlockDbKey(cid, pos)
			value := make([]byte, 4)
			binary.LittleEndian.PutUint32(value, uint32(w))
			if err := bkt.Put(key, value); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("flush error: %v", err)
	}
}

func unpackBlock(longs []int64, index int, bitsPerBlock int) int {
	blocksPerLong := 64 / bitsPerBlock
	longIndex := index / blocksPerLong
	if longIndex >= len(longs) {
		return -1
	}
	bitOffset := (index % blocksPerLong) * bitsPerBlock
	mask := int64((1 << bitsPerBlock) - 1)
	return int((longs[longIndex] >> bitOffset) & mask)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case byte:
		return int(int8(val))
	case int16:
		return int(val)
	case int32:
		return int(val)
	case int64:
		return int(val)
	default:
		return 0
	}
}

func encodeBlockDbKey(cid, bid Vec3) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, [...]int32{int32(cid.X), int32(cid.Z)})
	binary.Write(buf, binary.LittleEndian, [...]int32{int32(bid.X), int32(bid.Y), int32(bid.Z)})
	return buf.Bytes()
}

func decodeBlockDbKey(b []byte) (Vec3, Vec3) {
	buf := bytes.NewBuffer(b)
	var arr [5]int32
	binary.Read(buf, binary.LittleEndian, &arr)
	cid := Vec3{int(arr[0]), 0, int(arr[1])}
	bid := Vec3{int(arr[2]), int(arr[3]), int(arr[4])}
	return cid, bid
}

func parseNBTFull(r *bytes.Reader) map[string]interface{} {
	tagType, err := r.ReadByte()
	if err != nil || tagType != tagCompound {
		return nil
	}
	readString(r)
	return readCompound(r)
}

func readCompound(r *bytes.Reader) map[string]interface{} {
	m := make(map[string]interface{})
	for {
		tagType, err := r.ReadByte()
		if err != nil || tagType == tagEnd {
			return m
		}
		name := readString(r)
		m[name] = readPayload(r, tagType)
	}
}

func readPayload(r *bytes.Reader, tagType byte) interface{} {
	switch tagType {
	case tagByte:
		v, _ := r.ReadByte()
		return v
	case tagShort:
		var v int16
		binary.Read(r, binary.BigEndian, &v)
		return v
	case tagInt:
		var v int32
		binary.Read(r, binary.BigEndian, &v)
		return v
	case tagLong:
		var v int64
		binary.Read(r, binary.BigEndian, &v)
		return v
	case tagFloat:
		var v float32
		binary.Read(r, binary.BigEndian, &v)
		return v
	case tagDouble:
		var v float64
		binary.Read(r, binary.BigEndian, &v)
		return v
	case tagByteArray:
		var l int32
		binary.Read(r, binary.BigEndian, &l)
		buf := make([]byte, l)
		r.Read(buf)
		return buf
	case tagString:
		return readString(r)
	case tagList:
		lt, _ := r.ReadByte()
		var l int32
		binary.Read(r, binary.BigEndian, &l)
		list := make([]interface{}, l)
		for i := int32(0); i < l; i++ {
			list[i] = readPayload(r, lt)
		}
		return list
	case tagCompound:
		return readCompound(r)
	case tagIntArray:
		var l int32
		binary.Read(r, binary.BigEndian, &l)
		arr := make([]int32, l)
		binary.Read(r, binary.BigEndian, &arr)
		return arr
	case tagLongArray:
		var l int32
		binary.Read(r, binary.BigEndian, &l)
		arr := make([]int64, l)
		binary.Read(r, binary.BigEndian, &arr)
		return arr
	default:
		return nil
	}
}

func readString(r *bytes.Reader) string {
	var l uint16
	binary.Read(r, binary.BigEndian, &l)
	buf := make([]byte, l)
	r.Read(buf)
	return string(buf)
}

func ParseLevelDat(path string) (spawnX, spawnY, spawnZ int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, 0, err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("gzip: %w", err)
	}
	defer gr.Close()

	raw, err := io.ReadAll(gr)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("read: %w", err)
	}

	r := bytes.NewReader(raw)
	root := parseNBTFull(r)
	if root == nil {
		return 0, 0, 0, fmt.Errorf("invalid NBT")
	}

	data, ok := root["Data"]
	if !ok {
		return 0, 0, 0, fmt.Errorf("no Data compound")
	}
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return 0, 0, 0, fmt.Errorf("Data is not compound")
	}

	spawnX = toInt(dataMap["SpawnX"])
	spawnY = toInt(dataMap["SpawnY"])
	spawnZ = toInt(dataMap["SpawnZ"])
	return spawnX, spawnY, spawnZ, nil
}

func ParseLevelDatName(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gr.Close()

	raw, err := io.ReadAll(gr)
	if err != nil {
		return "", err
	}

	r := bytes.NewReader(raw)
	root := parseNBTFull(r)
	if root == nil {
		return "", fmt.Errorf("invalid NBT")
	}

	data, ok := root["Data"]
	if !ok {
		return "", fmt.Errorf("no Data compound")
	}
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("Data is not compound")
	}

	name, ok := dataMap["LevelName"]
	if !ok {
		return "", fmt.Errorf("no LevelName")
	}
	return name.(string), nil
}
