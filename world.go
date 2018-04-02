package main

import (
	"sync"

	"github.com/go-gl/mathgl/mgl32"
)

type World struct {
	chunks sync.Map
}

func NewWorld() *World {
	return &World{}
}

func (w *World) Collide(pos mgl32.Vec3) (mgl32.Vec3, bool) {
	x, y, z := pos.X(), pos.Y(), pos.Z()
	nx, ny, nz := round(pos.X()), round(pos.Y()), round(pos.Z())
	const pad = 0.25

	head := Vec3{int(nx), int(ny), int(nz)}
	foot := head.Down()

	stop := false
	for _, b := range []Vec3{foot, head} {
		if IsObstacle(w.Block(b.Left())) && x < nx && nx-x > pad {
			x = nx - pad
		}
		if IsObstacle(w.Block(b.Right())) && x > nx && x-nx > pad {
			x = nx + pad
		}
		if IsObstacle(w.Block(b.Down())) && y < ny && ny-y > pad {
			y = ny - pad
			stop = true
		}
		if IsObstacle(w.Block(b.Up())) && y > ny && y-ny > pad {
			y = ny + pad
			stop = true
		}
		if IsObstacle(w.Block(b.Back())) && z < nz && nz-z > pad {
			z = nz - pad
		}
		if IsObstacle(w.Block(b.Front())) && z > nz && z-nz > pad {
			z = nz + pad
		}
	}
	return mgl32.Vec3{x, y, z}, stop
}

func (w *World) HitTest(pos mgl32.Vec3, vec mgl32.Vec3) (*Vec3, *Vec3) {
	var (
		maxLen = float32(8.0)
		step   = float32(0.125)

		block, prev Vec3
		pprev       *Vec3
	)

	for len := float32(0); len < maxLen; len += step {
		block = NearBlock(pos.Add(vec.Mul(len)))
		if prev != block && w.HasBlock(block) {
			return &block, pprev
		}
		prev = block
		pprev = &prev
	}
	return nil, nil
}

func (w *World) Block(id Vec3) int {
	chunk := w.BlockChunk(id)
	if chunk == nil {
		return -1
	}
	return chunk.Block(id)
}

func (w *World) BlockChunk(block Vec3) *Chunk {
	cid := block.Chunkid()
	chunk, ok := w.chunks.Load(cid)
	if !ok {
		return nil
	}
	return chunk.(*Chunk)
}

func IsPlant(tp int) bool {
	if tp >= 17 && tp <= 31 {
		return true
	}
	return false
}

func IsTransparent(tp int) bool {
	if IsPlant(tp) {
		return true
	}
	switch tp {
	case -1, 0, 10, 15:
		return true
	default:
		return false
	}
}

func IsObstacle(tp int) bool {
	if IsPlant(tp) {
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

func (w *World) HasBlock(id Vec3) bool {
	tp := w.Block(id)
	return tp != -1 && tp != 0
}

func (w *World) Chunk(id Vec3) *Chunk {
	p, ok := w.chunks.Load(id)
	if ok {
		return p.(*Chunk)
	}
	chunk := NewChunk(w, id)
	blocks := makeChunkMap(id)
	for block, tp := range blocks {
		chunk.Add(block, tp)
	}
	w.chunks.Store(id, chunk)
	return chunk
}

func (w *World) Chunks(ids []Vec3) []*Chunk {
	ch := make(chan *Chunk)
	var chunks []*Chunk
	for _, id := range ids {
		id := id
		go func() {
			ch <- w.Chunk(id)
		}()
	}
	for range ids {
		chunk := <-ch
		if chunk != nil {
			chunks = append(chunks, chunk)
		}
	}
	return chunks
}

func makeChunkMap(cid Vec3) map[Vec3]int {
	const (
		grassBlock = 1
		sandBlock  = 2
		grass      = 17
		leaves     = 15
		wood       = 5
	)
	m := make(map[Vec3]int)
	p, q := cid.X, cid.Z
	for dx := 0; dx < ChunkWidth; dx++ {
		for dz := 0; dz < ChunkWidth; dz++ {
			x, z := p*ChunkWidth+dx, q*ChunkWidth+dz
			f := noise2(float32(x)*0.01, float32(z)*0.01, 4, 0.5, 2)
			g := noise2(float32(-x)*0.01, float32(-z)*0.01, 2, 0.9, 2)
			mh := int(g*32 + 16)
			h := int(f * float32(mh))
			w := grassBlock
			if h <= 12 {
				h = 12
				w = sandBlock
			}
			// grass and sand
			for y := 0; y < h; y++ {
				m[Vec3{x, y, z}] = w
			}

			// flowers
			if w == grassBlock {
				if noise2(-float32(x)*0.1, float32(z)*0.1, 4, 0.8, 2) > 0.6 {
					m[Vec3{x, h, z}] = grass
				}
				if noise2(float32(x)*0.05, float32(-z)*0.05, 4, 0.8, 2) > 0.7 {
					w := 18 + int(noise2(float32(x)*0.1, float32(z)*0.1, 4, 0.8, 2)*7)
					m[Vec3{x, h, z}] = w
				}
			}

			// tree
			if w == 1 {
				ok := true
				if dx-4 < 0 || dz-4 < 0 ||
					dx+4 > ChunkWidth || dz+4 > ChunkWidth {
					ok = false
				}
				if ok && noise2(float32(x), float32(z), 6, 0.5, 2) > 0.79 {
					for y := h + 3; y < h+8; y++ {
						for ox := -3; ox <= 3; ox++ {
							for oz := -3; oz <= 3; oz++ {
								d := ox*ox + oz*oz + (y-h-4)*(y-h-4)
								if d < 11 {
									m[Vec3{x + ox, y, z + oz}] = leaves
								}
							}
						}
					}
					for y := h; y < h+7; y++ {
						m[Vec3{x, y, z}] = wood
					}
				}
			}

			// cloud
			for y := 64; y < 72; y++ {
				if noise3(float32(x)*0.01, float32(y)*0.1, float32(z)*0.01, 8, 0.5, 2) > 0.69 {
					m[Vec3{x, y, z}] = 16
				}
			}
		}
	}
	return m
}
