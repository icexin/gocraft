package main

import (
	"log"

	"github.com/faiface/glhf"
	"github.com/faiface/mainthread"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/icexin/gocraft-server/proto"
)

type PlayerState struct {
	X, Y, Z float32
	Rx, Ry  float32
}

type playerState struct {
	PlayerState
	time float64
}

type Player struct {
	s1, s2 playerState

	shader *glhf.Shader
	mesh   *Mesh
}

// 线性插值计算玩家位置
func (p *Player) computeMat() mgl32.Mat4 {
	t1 := p.s2.time - p.s1.time
	t2 := glfw.GetTime() - p.s2.time
	t := min(float32(t2/t1), 1)

	x := mix(p.s1.X, p.s2.X, t)
	y := mix(p.s1.Y, p.s2.Y, t)
	z := mix(p.s1.Z, p.s2.Z, t)
	rx := mix(p.s1.Rx, p.s2.Rx, t)
	ry := mix(p.s1.Ry, p.s2.Ry, t)

	front := mgl32.Vec3{
		cos(radian(ry)) * cos(radian(rx)),
		sin(radian(ry)),
		cos(radian(ry)) * sin(radian(rx)),
	}.Normalize()
	right := front.Cross(mgl32.Vec3{0, 1, 0})
	up := right.Cross(front).Normalize()
	pos := mgl32.Vec3{x, y, z}
	return mgl32.LookAtV(pos, pos.Add(front), up).Inv()
}

func (p *Player) UpdateState(s playerState) {
	p.s1, p.s2 = p.s2, s
}

func (p *Player) Draw(mat mgl32.Mat4) {
	mat = mat.Mul4(p.computeMat())

	p.shader.SetUniformAttr(0, mat)
	p.mesh.Draw()
}

func (p *Player) Release() {
	p.mesh.Release()
}

type PlayerRender struct {
	shader  *glhf.Shader
	texture *glhf.Texture
	players map[int32]*Player
}

func NewPlayerRender() (*PlayerRender, error) {
	var (
		err error
	)
	img, rect, err := loadImage(*texturePath)
	if err != nil {
		return nil, err
	}

	r := &PlayerRender{
		players: make(map[int32]*Player),
	}
	mainthread.Call(func() {
		r.shader, err = glhf.NewShader(glhf.AttrFormat{
			glhf.Attr{Name: "pos", Type: glhf.Vec3},
			glhf.Attr{Name: "tex", Type: glhf.Vec2},
			glhf.Attr{Name: "normal", Type: glhf.Vec3},
		}, glhf.AttrFormat{
			glhf.Attr{Name: "matrix", Type: glhf.Mat4},
		}, playerVertexSource, playerFragmentSource)

		if err != nil {
			return
		}
		r.texture = glhf.NewTexture(rect.Dx(), rect.Dy(), false, img)

	})
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *PlayerRender) UpdateOrAdd(id int32, s proto.PlayerState) {
	state := playerState{
		PlayerState: PlayerState{
			X:  s.X,
			Y:  s.Y,
			Z:  s.Z,
			Rx: s.Rx,
			Ry: s.Ry,
		},
		time: glfw.GetTime(),
	}

	p, ok := r.players[id]
	if !ok {
		log.Printf("add new player %d", id)
		cubeData := makeCubeData([]float32{}, [...]bool{true, true, true, true, true, true}, Vec3{0, 0, 0}, tex.Texture(64))
		var mesh *Mesh
		mainthread.Call(func() {
			mesh = NewMesh(r.shader, cubeData)
		})
		p = &Player{
			shader: r.shader,
			mesh:   mesh,
		}
		r.players[id] = p
		p.s1 = state
	}
	p.UpdateState(state)
}

func (r *PlayerRender) Remove(id int32) {
	log.Printf("remove player %d", id)
	p, ok := r.players[id]
	if ok {
		mainthread.CallNonBlock(func() {
			p.Release()
		})
	}
	delete(r.players, id)

}

func (r *PlayerRender) Draw() {
	mat := game.blockRender.get3dmat()
	r.shader.Begin()
	r.texture.Begin()
	for _, p := range r.players {
		p.Draw(mat)
	}
	r.texture.End()
	r.shader.End()
}
