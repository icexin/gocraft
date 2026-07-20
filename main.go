package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
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

	game *Game
)

type Game struct {
	win *glfw.Window

	camera   *Camera
	lx, ly   float64
	vy       float32
	prevtime float64

	blockRender  *BlockRender
	lineRender   *LineRender
	playerRender *PlayerRender

	world   *World
	itemidx int
	item    int
	fps     FPS

	exclusiveMouse bool
	closed         bool

	history   *EditHistory
	hotbar    *Hotbar
	showDebug bool
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
	game.history = NewEditHistory(50)
	game.hotbar = NewHotbar()

	mainthread.Call(func() {
		win := initGL(w, h)
		win.SetMouseButtonCallback(game.onMouseButtonCallback)
		win.SetCursorPosCallback(game.onCursorPosCallback)
		win.SetFramebufferSizeCallback(game.onFrameBufferSizeCallback)
		win.SetKeyCallback(game.onKeyCallback)
		game.win = win
	})
	game.world = NewWorld()
	game.camera = NewCamera(mgl32.Vec3{5, castleBaseY + 5, 20})
	game.blockRender, err = NewBlockRender()
	if err != nil {
		return nil, err
	}
	mainthread.Call(func() {
		game.blockRender.UpdateItem(game.item)
	})
	game.lineRender, err = NewLineRender()
	if err != nil {
		return nil, err
	}
	game.playerRender, err = NewPlayerRender()
	if err != nil {
		return nil, err
	}
	gameClock = NewGameClock(600)
	go game.blockRender.UpdateLoop()
	go game.syncPlayerLoop()
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
	g.blockRender.DirtyChunk(cid)
	neighbors := []Vec3{id.Left(), id.Right(), id.Front(), id.Back()}
	for _, neighbor := range neighbors {
		chunkid := neighbor.Chunkid()
		if chunkid != cid {
			g.blockRender.DirtyChunk(chunkid)
		}
	}
}

func (g *Game) onMouseButtonCallback(win *glfw.Window, button glfw.MouseButton, action glfw.Action, mod glfw.ModifierKey) {
	if !g.exclusiveMouse {
		g.setExclusiveMouse(true)
		return
	}
	head := NearBlock(g.camera.Pos())
	foot := head.Down()
	block, prev := g.world.HitTest(g.camera.Pos(), g.camera.Front())
	if button == glfw.MouseButton2 && action == glfw.Press {
		if prev != nil && *prev != head && *prev != foot {
			oldType := g.world.Block(*prev)
			g.history.Push(EditAction{Position: *prev, OldType: oldType, NewType: g.item})
			g.world.UpdateBlock(*prev, g.item)
			g.dirtyBlock(*prev)
			go ClientUpdateBlock(*prev, g.item)
		}
	}
	if button == glfw.MouseButton1 && action == glfw.Press {
		if block != nil {
			oldType := g.world.Block(*block)
			g.history.Push(EditAction{Position: *block, OldType: oldType, NewType: 0})
			g.world.UpdateBlock(*block, 0)
			g.dirtyBlock(*block)
			go ClientUpdateBlock(*block, 0)
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
	g.camera.OnAngleChange(float32(dx), float32(dy))
}

func (g *Game) onKeyCallback(win *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if action != glfw.Press {
		return
	}
	switch key {
	case glfw.KeyZ:
		if mods&glfw.ModControl != 0 {
			if action, ok := g.history.Pop(); ok {
				g.world.UpdateBlock(action.Position, action.OldType)
				g.dirtyBlock(action.Position)
				go ClientUpdateBlock(action.Position, action.OldType)
			}
		}
	case glfw.KeyTab:
		g.camera.FlipFlying()
	case glfw.KeySpace:
		block := g.CurrentBlockid()
		if g.world.HasBlock(Vec3{block.X, block.Y - 2, block.Z}) {
			g.vy = 8
		}
	case glfw.KeyE:
		g.hotbar.NextSlot()
		g.item = g.hotbar.SelectedItem()
		g.blockRender.UpdateItem(g.item)
	case glfw.KeyR:
		g.hotbar.PrevSlot()
		g.item = g.hotbar.SelectedItem()
		g.blockRender.UpdateItem(g.item)
	case glfw.KeyF3:
		g.showDebug = !g.showDebug
	case glfw.Key1, glfw.Key2, glfw.Key3, glfw.Key4, glfw.Key5,
		glfw.Key6, glfw.Key7, glfw.Key8, glfw.Key9:
		slot := int(key - glfw.Key1)
		g.hotbar.Select(slot)
		g.item = g.hotbar.SelectedItem()
		g.blockRender.UpdateItem(g.item)
	}
}

func (g *Game) handleKeyInput(dt float64) {
	speed := float32(0.1)
	if g.win.GetKey(glfw.KeyEscape) == glfw.Press {
		g.setExclusiveMouse(false)
	}

	wPressed := g.win.GetKey(glfw.KeyW) == glfw.Press
	shiftHeld := g.win.GetKey(glfw.KeyLeftShift) == glfw.Press
	ctrlHeld := g.win.GetKey(glfw.KeyLeftControl) == glfw.Press

	if ctrlHeld && wPressed {
		g.camera.SetMovementMode(ModeSprint)
	} else if shiftHeld {
		g.camera.SetMovementMode(ModeCrouch)
	} else {
		g.camera.SetMovementMode(ModeWalk)
	}

	g.camera.UpdateFOV(float32(dt))

	prev := g.camera.Pos()
	if wPressed {
		g.camera.OnMoveChange(MoveForward, speed)
	}
	if g.win.GetKey(glfw.KeyS) == glfw.Press {
		g.camera.OnMoveChange(MoveBackward, speed)
	}
	if g.win.GetKey(glfw.KeyA) == glfw.Press {
		g.camera.OnMoveChange(MoveLeft, speed)
	}
	if g.win.GetKey(glfw.KeyD) == glfw.Press {
		g.camera.OnMoveChange(MoveRight, speed)
	}
	pos := g.camera.Pos()
	stop := false
	if !g.camera.Flying() {
		g.vy -= float32(dt * 20)
		if g.vy < -50 {
			g.vy = -50
		}
		pos = mgl32.Vec3{pos.X(), pos.Y() + g.vy*float32(dt), pos.Z()}
	}

	if g.camera.GetMovementMode() == ModeCrouch && !g.camera.Flying() {
		pos, stop = CollideCrouch(g.world, prev, pos)
	} else {
		pos, stop = CollideStepped(g.world, prev, pos)
	}
	if stop {
		g.vy = 0
	}
	g.camera.SetPos(pos)
}

func (g *Game) CurrentBlockid() Vec3 {
	pos := g.camera.Pos()
	return NearBlock(pos)
}

func (g *Game) ShouldClose() bool {
	return g.closed
}

func (g *Game) renderStat() {
	g.fps.Update()
	p := g.camera.Pos()
	cid := NearBlock(p).Chunkid()
	stat := g.blockRender.Stat()
	slot := g.hotbar.Selected() + 1
	if g.showDebug {
		block, _ := g.world.HitTest(g.camera.Pos(), g.camera.Front())
		lookInfo := ""
		if block != nil {
			lookInfo = fmt.Sprintf(" | Look: %d @ %v", g.world.Block(*block), *block)
		}
		title := fmt.Sprintf("[%.1f %.1f %.1f] C:%v [%d/%d F:%d] FPS:%d Slot:%d%s",
			p.X(), p.Y(), p.Z(), cid, stat.RendingChunks, stat.CacheChunks, stat.Faces, g.fps.Fps(), slot, lookInfo)
		g.win.SetTitle(title)
	} else {
		title := fmt.Sprintf("gocraft [%d/%d] FPS:%d Slot:%d", stat.RendingChunks, stat.CacheChunks, g.fps.Fps(), slot)
		g.win.SetTitle(title)
	}
}

func (g *Game) syncPlayerLoop() {
	tick := time.NewTicker(time.Second / 10)
	for range tick.C {
		ClientUpdatePlayerState(g.camera.State())
	}
}

func (g *Game) Update() {
	mainthread.Call(func() {
		var dt float64
		now := glfw.GetTime()
		dt = now - g.prevtime
		g.prevtime = now
		if dt > 0.02 {
			dt = 0.02
		}

		g.handleKeyInput(dt)

		sky := SkyColor()
		gl.ClearColor(sky.X(), sky.Y(), sky.Z(), 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		g.blockRender.Draw()
		g.lineRender.Draw()
		g.playerRender.Draw()

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

	err = InitStore()
	if err != nil {
		log.Panic(err)
	}
	defer store.Close()

	err = InitClient()
	if err != nil {
		log.Panic(err)
	}
	if client != nil {
		defer client.Close()
	}

	game, err = NewGame(800, 600)
	if err != nil {
		log.Panic(err)
	}

	game.camera.Restore(store.GetPlayerState())
	stopCh := SetupSignalHandler()
	go func() {
		<-stopCh
		log.Println("received shutdown signal")
		game.closed = true
	}()
	tick := time.Tick(time.Second / 60)
	for !game.ShouldClose() {
		<-tick
		game.Update()
	}
	store.UpdatePlayerState(game.camera.State())
}

func SetupSignalHandler() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	return ch
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	flag.Parse()
	go func() {
		if *pprofPort != "" {
			log.Fatal(http.ListenAndServe(*pprofPort, nil))
		}
	}()
	mainthread.Run(run)
}
