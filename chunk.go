package main

import (
	"encoding/gob"
	"io"
	"log"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/go-gl/mathgl/mgl32"
)

const (
	ChunkWidth = 32
)

type Vec3 struct {
	X, Y, Z int
}

func (v Vec3) Left() Vec3 {
	return Vec3{v.X - 1, v.Y, v.Z}
}
func (v Vec3) Right() Vec3 {
	return Vec3{v.X + 1, v.Y, v.Z}
}
func (v Vec3) Up() Vec3 {
	return Vec3{v.X, v.Y + 1, v.Z}
}
func (v Vec3) Down() Vec3 {
	return Vec3{v.X, v.Y - 1, v.Z}
}
func (v Vec3) Front() Vec3 {
	return Vec3{v.X, v.Y, v.Z + 1}
}
func (v Vec3) Back() Vec3 {
	return Vec3{v.X, v.Y, v.Z - 1}
}
func (v Vec3) Chunkid() Vec3 {
	return Vec3{
		int(math.Floor(float64(v.X) / ChunkWidth)),
		0,
		int(math.Floor(float64(v.Z) / ChunkWidth)),
	}
}

func (v Vec3) ChunkidString() string {
	return strconv.Itoa(v.X) + "_" + strconv.Itoa(v.Z)
}

func NearBlock(pos mgl32.Vec3) Vec3 {
	return Vec3{
		int(round(pos.X())),
		int(round(pos.Y())),
		int(round(pos.Z())),
	}
}

type Chunk struct {
	id     Vec3
	world  *World
	Blocks sync.Map

	Version int64
}

func NewChunk(w *World, id Vec3) *Chunk {
	c := &Chunk{
		id:      id,
		world:   w,
		Version: time.Now().Unix(),
	}
	return c
}

func (c *Chunk) SaveToWriter(writer io.Writer) {
	enc := gob.NewEncoder(writer)
	enc.Encode(c.id)
	enc.Encode(c.Version)
	simpleBlockMap := make(map[Vec3]int)
	c.Blocks.Range(func(key, block interface{}) bool {
		simpleBlockMap[key.(Vec3)] = block.(int)
		return true
	})

	enc.Encode(simpleBlockMap)
}

func (c *Chunk) LoadFromReader(reader io.Reader) {
	dec := gob.NewDecoder(reader)
	dec.Decode(&c.id)
	dec.Decode(&c.Version)
	var simpleBlockMap map[Vec3]int
	dec.Decode(&simpleBlockMap)
	c.Blocks = sync.Map{}
	for k, v := range simpleBlockMap {
		c.Blocks.Store(k, v)
	}

}

func (c *Chunk) UpdateVersion() {
	c.Version = time.Now().UnixNano() / int64(time.Millisecond)
}

func (c *Chunk) Id() Vec3 {
	return c.id
}

func (c *Chunk) Block(id Vec3) int {
	if id.Chunkid() != c.id {
		log.Panicf("id %v chunk %v", id, c.id)
	}
	w, ok := c.Blocks.Load(id)
	if ok {
		return w.(int)
	}
	return 0
}

func (c *Chunk) Add(id Vec3, w int) {
	if id.Chunkid() != c.id {
		log.Panicf("id %v chunk %v", id, c.id)
	}
	c.Blocks.Store(id, w)
	c.UpdateVersion()
}

func (c *Chunk) Del(id Vec3) {
	if id.Chunkid() != c.id {
		log.Panicf("id %v chunk %v", id, c.id)
	}
	c.Blocks.Delete(id)
	c.UpdateVersion()
}

func (c *Chunk) RangeBlocks(f func(id Vec3, w int)) {
	c.Blocks.Range(func(key, value interface{}) bool {
		f(key.(Vec3), value.(int))
		return true
	})
}
