package main

import "fmt"

const (
	HotbarSlots    = 9
	FontGlyphSize  = 8
	FontAtlasCols  = 16
	FontAtlasRows  = 6
	FontStartChar  = 32
)

type Hotbar struct {
	slots    [HotbarSlots]int
	selected int
}

func NewHotbar() *Hotbar {
	h := &Hotbar{}
	for i := 0; i < HotbarSlots && i < len(availableItems); i++ {
		h.slots[i] = availableItems[i]
	}
	return h
}

func (h *Hotbar) Select(slot int) {
	if slot >= 0 && slot < HotbarSlots {
		h.selected = slot
	}
}

func (h *Hotbar) Selected() int {
	return h.selected
}

func (h *Hotbar) SelectedItem() int {
	return h.slots[h.selected]
}

func (h *Hotbar) SetSlot(slot, itemID int) {
	if slot >= 0 && slot < HotbarSlots {
		h.slots[slot] = itemID
	}
}

func (h *Hotbar) GetSlot(slot int) int {
	if slot >= 0 && slot < HotbarSlots {
		return h.slots[slot]
	}
	return 0
}

func (h *Hotbar) NextSlot() {
	h.selected = (h.selected + 1) % HotbarSlots
}

func (h *Hotbar) PrevSlot() {
	h.selected--
	if h.selected < 0 {
		h.selected = HotbarSlots - 1
	}
}

type HUDState struct {
	HotbarItems   [HotbarSlots]int
	SelectedSlot  int
	ShowDebug     bool
	FPS           int
	PlayerPos     [3]float32
	ChunkID       Vec3
	LookingAt     *Vec3
	LookingAtType int
	ChunkCount    int
	FaceCount     int
}

func CollectHUDState(g *Game) HUDState {
	state := HUDState{
		SelectedSlot: g.hotbar.Selected(),
		FPS:          g.fps.Fps(),
		ChunkCount:   g.blockRender.Stat().CacheChunks,
		FaceCount:    g.blockRender.Stat().Faces,
		ShowDebug:    g.showDebug,
	}
	pos := g.camera.Pos()
	state.PlayerPos = [3]float32{pos.X(), pos.Y(), pos.Z()}
	state.ChunkID = NearBlock(pos).Chunkid()
	for i := 0; i < HotbarSlots; i++ {
		state.HotbarItems[i] = g.hotbar.GetSlot(i)
	}
	block, _ := g.world.HitTest(g.camera.Pos(), g.camera.Front())
	if block != nil {
		state.LookingAt = block
		state.LookingAtType = g.world.Block(*block)
	}
	return state
}

func FormatDebugOverlay(state HUDState) string {
	s := fmt.Sprintf("FPS: %d\nXYZ: %.1f / %.1f / %.1f\nChunk: %d, %d, %d\n",
		state.FPS,
		state.PlayerPos[0], state.PlayerPos[1], state.PlayerPos[2],
		state.ChunkID.X, state.ChunkID.Y, state.ChunkID.Z)
	if state.LookingAt != nil {
		s += fmt.Sprintf("Looking at: %d @ %d, %d, %d\n",
			state.LookingAtType,
			state.LookingAt.X, state.LookingAt.Y, state.LookingAt.Z)
	}
	s += fmt.Sprintf("Chunks: %d cached\nFaces: %d",
		state.ChunkCount, state.FaceCount)
	return s
}

func GlyphUV(char byte) (u, v, u2, v2 float32) {
	if char < FontStartChar || int(char-FontStartChar) >= FontAtlasCols*FontAtlasRows {
		char = '?'
	}
	idx := int(char - FontStartChar)
	col := idx % FontAtlasCols
	row := idx / FontAtlasCols
	atlasW := float32(FontAtlasCols * FontGlyphSize)
	atlasH := float32(FontAtlasRows * FontGlyphSize)
	u = float32(col*FontGlyphSize) / atlasW
	v = float32(row*FontGlyphSize) / atlasH
	u2 = float32((col+1)*FontGlyphSize) / atlasW
	v2 = float32((row+1)*FontGlyphSize) / atlasH
	return
}
