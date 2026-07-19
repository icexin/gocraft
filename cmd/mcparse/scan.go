package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func scanAllBlocks(regionDir string) {
	blockCounts := make(map[string]int)

	files, err := filepath.Glob(filepath.Join(regionDir, "r.*.mca"))
	if err != nil || len(files) == 0 {
		fmt.Println("no region files found")
		return
	}

	for _, f := range files {
		scanRegion(f, blockCounts)
	}

	type kv struct {
		name  string
		count int
	}
	var sorted []kv
	for k, v := range blockCounts {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	fmt.Printf("\n=== All unique block types (%d) ===\n", len(sorted))
	for _, s := range sorted {
		fmt.Printf("  %s: %d\n", s.name, s.count)
	}
}

func scanRegion(path string, blockCounts map[string]int) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 8192 {
		return
	}

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
			for _, p := range paletteList {
				pMap, ok := p.(map[string]interface{})
				if !ok {
					continue
				}
				if name, ok := pMap["Name"]; ok {
					blockCounts[name.(string)]++
				}
			}
		}
	}
}
