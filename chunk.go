package main

import (
	"log"
	"math"
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
	blocks sync.Map

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
	w, ok := c.blocks.Load(id)
	if ok {
		return w.(int)
	}
	return 0
}

func (c *Chunk) Add(id Vec3, w int) {
	if id.Chunkid() != c.id {
		log.Panicf("id %v chunk %v", id, c.id)
	}
	c.blocks.Store(id, w)
	c.UpdateVersion()
}

func (c *Chunk) Del(id Vec3) {
	if id.Chunkid() != c.id {
		log.Panicf("id %v chunk %v", id, c.id)
	}
	c.blocks.Delete(id)
	c.UpdateVersion()
}

func (c *Chunk) RangeBlocks(f func(id Vec3, w int)) {
	c.blocks.Range(func(key, value interface{}) bool {
		f(key.(Vec3), value.(int))
		return true
	})
}
