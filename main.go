package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "image/png"

	"net/http"
	_ "net/http/pprof"

	"github.com/faiface/mainthread"
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

var (
	pprofPort = flag.String("pprof", "", "http pprof port")
)

const savePath = "./game_save"

type Game struct {
	win *glfw.Window

	Camera   *Camera
	lx, ly   float64
	vy       float32
	prevtime float64

	blockRender *BlockRender
	lineRender  *LineRender

	world   *World
	itemidx int
	item    int
	fps     FPS

	exclusiveMouse bool
	closed         bool
}

func initGL(w, h int) *glfw.Window {
	err := glfw.Init()
	if err != nil {
		log.Fatal(err)
	}

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, gl.TRUE)

	win, err := glfw.CreateWindow(w, h, "gocraft", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	win.MakeContextCurrent()
	err = gl.Init()
	if err != nil {
		log.Fatal(err)
	}
	glfw.SwapInterval(1) // enable vsync
	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.CULL_FACE)
	return win
}

func NewGame(w, h int) (*Game, error) {
	var (
		err  error
		game *Game
	)
	game = new(Game)
	game.item = availableItems[0]

	mainthread.Call(func() {
		win := initGL(w, h)
		win.SetMouseButtonCallback(game.onMouseButtonCallback)
		win.SetCursorPosCallback(game.onCursorPosCallback)
		win.SetFramebufferSizeCallback(game.onFrameBufferSizeCallback)
		win.SetKeyCallback(game.onKeyCallback)
		game.win = win
	})
	game.world = NewWorld()
	game.Camera = NewCamera(mgl32.Vec3{0, 16, 0})
	game.blockRender, err = NewBlockRender(game)
	if err != nil {
		return nil, err
	}
	mainthread.Call(func() {
		game.blockRender.UpdateItem(game.item)
	})
	game.lineRender, err = NewLineRender(game)
	if err != nil {
		return nil, err
	}
	go game.blockRender.UpdateLoop()
	return game, nil
}

func (g *Game) setExclusiveMouse(exclusive bool) {
	if exclusive {
		g.win.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	} else {
		g.win.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
	}
	g.exclusiveMouse = exclusive
}

func (g *Game) dirtyBlock(id Vec3) {
	cid := id.Chunkid()
	neighbors := []Vec3{id.Left(), id.Right(), id.Front(), id.Back()}
	for _, neighbor := range neighbors {
		chunkid := neighbor.Chunkid()
		if chunkid != cid {
			g.world.Chunk(chunkid).UpdateVersion()
		}
	}
}

func (g *Game) onMouseButtonCallback(win *glfw.Window, button glfw.MouseButton, action glfw.Action, mod glfw.ModifierKey) {
	if !g.exclusiveMouse {
		g.setExclusiveMouse(true)
		return
	}
	head := NearBlock(g.Camera.Pos())
	foot := head.Down()
	block, prev := g.world.HitTest(g.Camera.Pos(), g.Camera.Front())
	if button == glfw.MouseButton2 && action == glfw.Press {
		if prev != nil && *prev != head && *prev != foot {
			chunk := g.world.BlockChunk(*prev)
			chunk.Add(*prev, g.item)
			g.dirtyBlock(*prev)
		}
	}
	if button == glfw.MouseButton1 && action == glfw.Press {
		if block != nil {
			chunk := g.world.BlockChunk(*block)
			chunk.Del(*block)
			g.dirtyBlock(*block)
		}
	}
}

func (g *Game) onFrameBufferSizeCallback(window *glfw.Window, width, height int) {
	gl.Viewport(0, 0, int32(width), int32(height))
}

func (g *Game) onCursorPosCallback(win *glfw.Window, xpos float64, ypos float64) {
	if !g.exclusiveMouse {
		return
	}
	if g.lx == 0 && g.ly == 0 {
		g.lx, g.ly = xpos, ypos
		return
	}
	dx, dy := xpos-g.lx, g.ly-ypos
	g.lx, g.ly = xpos, ypos
	g.Camera.OnAngleChange(float32(dx), float32(dy))
}

func (g *Game) onKeyCallback(win *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {

	if action != glfw.Press {
		return
	}
	switch key {
	case glfw.KeyTab:
		g.Camera.FlipFlying()
	case glfw.KeySpace:
		block := g.CurrentBlockid()
		if g.world.HasBlock(Vec3{block.X, block.Y - 2, block.Z}) {
			g.vy = 8
		}
	case glfw.KeyE:
		g.itemidx = (1 + g.itemidx) % len(availableItems)
		g.item = availableItems[g.itemidx]
		g.blockRender.UpdateItem(g.item)
	case glfw.KeyR:
		g.itemidx--
		if g.itemidx < 0 {
			g.itemidx = len(availableItems) - 1
		}
		g.item = availableItems[g.itemidx]
		g.blockRender.UpdateItem(g.item)
	case glfw.KeyK:
		g.saveGame()
	case glfw.KeyL:
		g.loadGame()
	}
}
func (g *Game) saveGame() {
	fmt.Println("Saving game...")

	folderError := os.MkdirAll(savePath, 0777)
	if folderError != nil {
		fmt.Printf("Error while creating save directory: %v\n", folderError)
		return
	}
	file, err := os.Create(path.Join(savePath, "./game.dat"))
	if err != nil {
		fmt.Printf("Error while opening save file: %v\n", err)
		return
	}
	defer file.Close()
	encoder := gob.NewEncoder(file)
	encodeErr := encoder.Encode(g)
	if encodeErr != nil {
		fmt.Printf("Error while saving data: %v\n", encodeErr)
		return
	}
	g.world.chunks.Range(func(key, value interface{}) bool {
		chunkFile, fileErr := os.Create(path.Join(savePath, key.(Vec3).ChunkidString()+".chunk.dat"))
		if fileErr != nil {
			fmt.Printf("Error while saving chunk data: %v\n", encodeErr)
			return false
		}
		defer chunkFile.Close()
		value.(*Chunk).SaveToWriter(chunkFile)
		return true
	})
	fmt.Println("Game saved!")
}
func (g *Game) loadGame() {
	fmt.Println("Loading game...")
	file, err := os.Open(path.Join(savePath, "./game.dat"))
	if err != nil {
		fmt.Printf("Error while opening save file: %v\n", err)
		return
	}
	decoder := gob.NewDecoder(file)
	decodeErr := decoder.Decode(&g)
	if decodeErr != nil {
		fmt.Printf("Error while decoding data: %v\n", decodeErr)
		return
	}
	g.world.chunks = sync.Map{}
	files, dirErr := ioutil.ReadDir(savePath)
	if dirErr != nil {
		fmt.Printf("Error while opening save file: %v\n", err)
		return
	}
	for _, fi := range files {
		if !fi.IsDir() && strings.HasSuffix(fi.Name(), ".chunk.dat") {
			chunkFile, err := os.Open(path.Join(savePath, fi.Name()))
			if err != nil {
				fmt.Printf("Error while opening save file: %v\n", err)
				return
			}
			parsedName := strings.Split(strings.Split(fi.Name(), ".")[0], "_")
			x, convErr := strconv.Atoi(parsedName[0])
			z, convErr := strconv.Atoi(parsedName[1])
			fmt.Printf("name=%v x=%v z=%v\n", fi.Name(), x, z)
			if convErr != nil {
				fmt.Printf("Error while parsing chunk id: %v\n", convErr)
				return
			}
			loadedChunk := NewChunk(g.world, Vec3{x, 0, z})
			loadedChunk.LoadFromReader(chunkFile)
			g.world.chunks.Store(Vec3{x, 0, z}, loadedChunk)
		}
	}
	fmt.Println("Game loaded!")
}
func (g *Game) handleKeyInput(dt float64) {
	speed := float32(0.1)
	if g.Camera.flying {
		speed = 0.2
	}
	if g.win.GetKey(glfw.KeyEscape) == glfw.Press {
		g.setExclusiveMouse(false)
	}
	if g.win.GetKey(glfw.KeyW) == glfw.Press {
		g.Camera.OnMoveChange(MoveForward, speed)
	}
	if g.win.GetKey(glfw.KeyS) == glfw.Press {
		g.Camera.OnMoveChange(MoveBackward, speed)
	}
	if g.win.GetKey(glfw.KeyA) == glfw.Press {
		g.Camera.OnMoveChange(MoveLeft, speed)
	}
	if g.win.GetKey(glfw.KeyD) == glfw.Press {
		g.Camera.OnMoveChange(MoveRight, speed)
	}
	pos := g.Camera.Pos()
	stop := false
	if !g.Camera.Flying() {
		g.vy -= float32(dt * 20)
		if g.vy < -50 {
			g.vy = -50
		}
		pos = mgl32.Vec3{pos.X(), pos.Y() + g.vy*float32(dt), pos.Z()}
	}

	pos, stop = g.world.Collide(pos)
	if stop {
		g.vy = 0
	}
	g.Camera.SetPos(pos)
}

func (g *Game) CurrentBlockid() Vec3 {
	pos := g.Camera.Pos()
	return NearBlock(pos)
}

func (g *Game) ShouldClose() bool {
	return g.closed
}

func (g *Game) renderStat() {
	g.fps.Update()
	p := g.Camera.Pos()
	cid := NearBlock(p).Chunkid()
	stat := g.blockRender.Stat()
	title := fmt.Sprintf("[%.2f %.2f %.2f] %v [%d/%d %d] %d", p.X(), p.Y(), p.Z(),
		cid, stat.RendingChunks, stat.CacheChunks, stat.Faces, g.fps.Fps())
	g.win.SetTitle(title)
}

func (g *Game) Update() {
	mainthread.Call(func() {
		var dt float64
		if g.prevtime == 0 {
			dt = 0
		}
		dt = glfw.GetTime() - g.prevtime
		g.prevtime = glfw.GetTime()
		if dt > 0.02 {
			dt = 0.02
		}

		g.handleKeyInput(dt)

		gl.ClearColor(0.57, 0.71, 0.77, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		g.blockRender.Draw()
		g.lineRender.Draw()

		g.renderStat()

		g.win.SwapBuffers()
		glfw.PollEvents()
		g.closed = g.win.ShouldClose()
	})
}

type FPS struct {
	lastUpdate time.Time
	cnt        int
	fps        int
}

func (f *FPS) Update() {
	f.cnt++
	now := time.Now()
	p := now.Sub(f.lastUpdate)
	if p >= time.Second {
		f.fps = int(float64(f.cnt) / p.Seconds())
		f.cnt = 0
		f.lastUpdate = now
	}
}

func (f *FPS) Fps() int {
	return f.fps
}

func run() {
	err := LoadTextureDesc()
	if err != nil {
		log.Fatal(err)
	}

	game, err := NewGame(800, 600)
	if err != nil {
		log.Fatal(err)
	}
	tick := time.Tick(time.Second / 60)
	for !game.ShouldClose() {
		<-tick
		game.Update()
	}
}

func main() {
	gob.Register(Game{})
	gob.Register(Camera{})
	gob.Register(Chunk{})
	gob.Register(World{})
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	flag.Parse()
	go func() {
		if *pprofPort != "" {
			log.Fatal(http.ListenAndServe(*pprofPort, nil))
		}
	}()
	mainthread.Run(run)
}
