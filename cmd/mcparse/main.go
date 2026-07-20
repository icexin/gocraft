package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/bits"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: mcparse <region.mca|region_dir> [--scan]")
	}

	if len(os.Args) >= 3 && os.Args[2] == "--scan" {
		scanAllBlocks(os.Args[1])
		return
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	if len(data) < 8192 {
		log.Fatal("file too small for anvil format")
	}

	for i := 0; i < 1024; i++ {
		offset := int(data[i*4])<<16 | int(data[i*4+1])<<8 | int(data[i*4+2])
		sectorCount := data[i*4+3]
		if offset == 0 && sectorCount == 0 {
			continue
		}

		chunkX := i % 32
		chunkZ := i / 32

		byteOffset := offset * 4096
		if byteOffset+5 > len(data) {
			continue
		}

		length := int(binary.BigEndian.Uint32(data[byteOffset : byteOffset+4]))
		compressionType := data[byteOffset+4]

		if byteOffset+4+length > len(data) {
			continue
		}

		if compressionType != 2 {
			continue
		}

		compressedData := data[byteOffset+5 : byteOffset+4+length]
		reader, err := zlib.NewReader(bytes.NewReader(compressedData))
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

		fmt.Printf("=== Chunk (%d, %d) - %d sections ===\n", chunkX, chunkZ, len(sectionList))
		for _, sec := range sectionList {
			secMap, ok := sec.(map[string]interface{})
			if !ok {
				continue
			}
			y := secMap["Y"]
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
			for i, p := range paletteList {
				pMap, ok := p.(map[string]interface{})
				if !ok {
					continue
				}
				if name, ok := pMap["Name"]; ok {
					blockNames[i] = name.(string)
				}
			}

			fmt.Printf("  Section Y=%v, palette(%d): %v\n", y, len(blockNames), blockNames)

			if longData, ok := bsMap["data"]; ok {
				longs := longData.([]int64)
				bitsPerBlock := max(4, bits.Len(uint(len(paletteList)-1)))
				totalBlocks := 16 * 16 * 16

				blockCounts := make(map[string]int)
				for bi := 0; bi < totalBlocks; bi++ {
					idx := unpackBlock(longs, bi, bitsPerBlock)
					if idx >= 0 && idx < len(blockNames) {
						blockCounts[blockNames[idx]]++
					}
				}

				for name, count := range blockCounts {
					if name != "minecraft:air" {
						fmt.Printf("    %s: %d\n", name, count)
					}
				}
			} else if len(paletteList) == 1 {
				fmt.Printf("    (entire section is %s)\n", blockNames[0])
			}
		}

		if chunkX == 0 && chunkZ == 0 {
			break
		}
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
