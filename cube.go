package main

const (
	sleft = iota
	sright
	sup
	sdown
	sfront
	sback
)

// show: left, right, up, down, front, back,
func makeCubeData(vertices []float32, show [6]bool, block Vec3, tex *BlockTexture) []float32 {
	l, r := tex.Left, tex.Right
	u, d := tex.Up, tex.Down
	f, b := tex.Front, tex.Back
	x, y, z := float32(block.X), float32(block.Y), float32(block.Z)
	if show[sleft] {
		vertices = append(vertices, []float32{
			// left
			x - 0.5, y - 0.5, z - 0.5, l[0][0], l[0][1], -1, 0, 0,
			x - 0.5, y - 0.5, z + 0.5, l[1][0], l[1][1], -1, 0, 0,
			x - 0.5, y + 0.5, z + 0.5, l[2][0], l[2][1], -1, 0, 0,
			x - 0.5, y + 0.5, z + 0.5, l[3][0], l[3][1], -1, 0, 0,
			x - 0.5, y + 0.5, z - 0.5, l[4][0], l[4][1], -1, 0, 0,
			x - 0.5, y - 0.5, z - 0.5, l[5][0], l[5][1], -1, 0, 0,
		}...)
	}
	if show[sright] {
		vertices = append(vertices, []float32{
			// right
			x + 0.5, y - 0.5, z + 0.5, r[0][0], r[0][1], 1, 0, 0,
			x + 0.5, y - 0.5, z - 0.5, r[1][0], r[1][1], 1, 0, 0,
			x + 0.5, y + 0.5, z - 0.5, r[2][0], r[2][1], 1, 0, 0,
			x + 0.5, y + 0.5, z - 0.5, r[3][0], r[3][1], 1, 0, 0,
			x + 0.5, y + 0.5, z + 0.5, r[4][0], r[4][1], 1, 0, 0,
			x + 0.5, y - 0.5, z + 0.5, r[5][0], r[5][1], 1, 0, 0,
		}...)
	}
	if show[sup] {
		vertices = append(vertices, []float32{
			// top
			x - 0.5, y + 0.5, z + 0.5, u[0][0], u[0][1], 0, 1, 0,
			x + 0.5, y + 0.5, z + 0.5, u[1][0], u[1][1], 0, 1, 0,
			x + 0.5, y + 0.5, z - 0.5, u[2][0], u[2][1], 0, 1, 0,
			x + 0.5, y + 0.5, z - 0.5, u[3][0], u[3][1], 0, 1, 0,
			x - 0.5, y + 0.5, z - 0.5, u[4][0], u[4][1], 0, 1, 0,
			x - 0.5, y + 0.5, z + 0.5, u[5][0], u[5][1], 0, 1, 0,
		}...)
	}

	if show[sdown] {
		vertices = append(vertices, []float32{
			// bottom
			x - 0.5, y - 0.5, z - 0.5, d[0][0], d[0][1], 0, -1, 0,
			x + 0.5, y - 0.5, z - 0.5, d[1][0], d[1][1], 0, -1, 0,
			x + 0.5, y - 0.5, z + 0.5, d[2][0], d[2][1], 0, -1, 0,
			x + 0.5, y - 0.5, z + 0.5, d[3][0], d[3][1], 0, -1, 0,
			x - 0.5, y - 0.5, z + 0.5, d[4][0], d[4][1], 0, -1, 0,
			x - 0.5, y - 0.5, z - 0.5, d[5][0], d[5][1], 0, -1, 0,
		}...)
	}

	if show[sfront] {
		vertices = append(vertices, []float32{
			// front
			x - 0.5, y - 0.5, z + 0.5, f[0][0], f[0][1], 0, 0, 1,
			x + 0.5, y - 0.5, z + 0.5, f[1][0], f[1][1], 0, 0, 1,
			x + 0.5, y + 0.5, z + 0.5, f[2][0], f[2][1], 0, 0, 1,
			x + 0.5, y + 0.5, z + 0.5, f[3][0], f[3][1], 0, 0, 1,
			x - 0.5, y + 0.5, z + 0.5, f[4][0], f[4][1], 0, 0, 1,
			x - 0.5, y - 0.5, z + 0.5, f[5][0], f[5][1], 0, 0, 1,
		}...)
	}

	if show[sback] {
		vertices = append(vertices, []float32{
			// back
			x + 0.5, y - 0.5, z - 0.5, b[0][0], b[0][1], 0, 0, -1,
			x - 0.5, y - 0.5, z - 0.5, b[1][0], b[1][1], 0, 0, -1,
			x - 0.5, y + 0.5, z - 0.5, b[2][0], b[2][1], 0, 0, -1,
			x - 0.5, y + 0.5, z - 0.5, b[3][0], b[3][1], 0, 0, -1,
			x + 0.5, y + 0.5, z - 0.5, b[4][0], b[4][1], 0, 0, -1,
			x + 0.5, y - 0.5, z - 0.5, b[5][0], b[5][1], 0, 0, -1,
		}...)
	}

	return vertices
}

func makeWireFrameData(vertices []float32, show [6]bool) []float32 {
	if show[sleft] {
		vertices = append(vertices, []float32{
			// left
			-0.5, -0.5, -0.5,
			-0.5, -0.5, +0.5,

			-0.5, -0.5, +0.5,
			-0.5, +0.5, +0.5,

			-0.5, +0.5, +0.5,
			-0.5, +0.5, -0.5,

			-0.5, +0.5, -0.5,
			-0.5, -0.5, -0.5,
		}...)
	}
	if show[sright] {
		vertices = append(vertices, []float32{
			// right
			+0.5, -0.5, +0.5,
			+0.5, -0.5, -0.5,

			+0.5, -0.5, -0.5,
			+0.5, +0.5, -0.5,

			+0.5, +0.5, -0.5,
			+0.5, +0.5, +0.5,

			+0.5, +0.5, +0.5,
			+0.5, -0.5, +0.5,
		}...)
	}

	if show[sup] {
		vertices = append(vertices, []float32{
			// top
			-0.5, +0.5, +0.5,
			+0.5, +0.5, +0.5,

			+0.5, +0.5, +0.5,
			+0.5, +0.5, -0.5,

			+0.5, +0.5, -0.5,
			-0.5, +0.5, -0.5,

			-0.5, +0.5, -0.5,
			-0.5, +0.5, +0.5,
		}...)
	}

	if show[sdown] {
		vertices = append(vertices, []float32{
			// bottom
			+0.5, -0.5, +0.5,
			-0.5, -0.5, +0.5,

			-0.5, -0.5, +0.5,
			-0.5, -0.5, -0.5,

			-0.5, -0.5, -0.5,
			+0.5, -0.5, -0.5,

			+0.5, -0.5, -0.5,
			+0.5, -0.5, +0.5,
		}...)
	}

	if show[sfront] {
		// z front
		vertices = append(vertices, []float32{
			-0.5, -0.5, +0.5,
			+0.5, -0.5, +0.5,

			+0.5, -0.5, +0.5,
			+0.5, +0.5, +0.5,

			+0.5, +0.5, +0.5,
			-0.5, +0.5, +0.5,

			-0.5, +0.5, +0.5,
			-0.5, -0.5, +0.5,
		}...)
	}

	if show[sback] {
		vertices = append(vertices, []float32{
			// back
			+0.5, -0.5, -0.5,
			-0.5, -0.5, -0.5,

			-0.5, -0.5, -0.5,
			-0.5, +0.5, -0.5,

			-0.5, +0.5, -0.5,
			+0.5, +0.5, -0.5,

			+0.5, +0.5, -0.5,
			+0.5, -0.5, -0.5,
		}...)
	}

	return vertices
}

func makePlantData(vertices []float32, show [6]bool, block Vec3, tex *BlockTexture) []float32 {
	l, r := tex.Left, tex.Right
	f, b := tex.Front, tex.Back
	x, y, z := float32(block.X), float32(block.Y), float32(block.Z)
	vertices = append(vertices, []float32{
		// left
		x, y - 0.5, z - 0.5, l[0][0], l[0][1], -1, 0, 0,
		x, y - 0.5, z + 0.5, l[1][0], l[1][1], -1, 0, 0,
		x, y + 0.5, z + 0.5, l[2][0], l[2][1], -1, 0, 0,
		x, y + 0.5, z + 0.5, l[3][0], l[3][1], -1, 0, 0,
		x, y + 0.5, z - 0.5, l[4][0], l[4][1], -1, 0, 0,
		x, y - 0.5, z - 0.5, l[5][0], l[5][1], -1, 0, 0,
	}...)
	vertices = append(vertices, []float32{
		// right
		x, y - 0.5, z + 0.5, r[0][0], r[0][1], 1, 0, 0,
		x, y - 0.5, z - 0.5, r[1][0], r[1][1], 1, 0, 0,
		x, y + 0.5, z - 0.5, r[2][0], r[2][1], 1, 0, 0,
		x, y + 0.5, z - 0.5, r[3][0], r[3][1], 1, 0, 0,
		x, y + 0.5, z + 0.5, r[4][0], r[4][1], 1, 0, 0,
		x, y - 0.5, z + 0.5, r[5][0], r[5][1], 1, 0, 0,
	}...)

	vertices = append(vertices, []float32{
		// front
		x - 0.5, y - 0.5, z, f[0][0], f[0][1], 0, 0, 1,
		x + 0.5, y - 0.5, z, f[1][0], f[1][1], 0, 0, 1,
		x + 0.5, y + 0.5, z, f[2][0], f[2][1], 0, 0, 1,
		x + 0.5, y + 0.5, z, f[3][0], f[3][1], 0, 0, 1,
		x - 0.5, y + 0.5, z, f[4][0], f[4][1], 0, 0, 1,
		x - 0.5, y - 0.5, z, f[5][0], f[5][1], 0, 0, 1,
	}...)

	vertices = append(vertices, []float32{
		// back
		x + 0.5, y - 0.5, z, b[0][0], b[0][1], 0, 0, -1,
		x - 0.5, y - 0.5, z, b[1][0], b[1][1], 0, 0, -1,
		x - 0.5, y + 0.5, z, b[2][0], b[2][1], 0, 0, -1,
		x - 0.5, y + 0.5, z, b[3][0], b[3][1], 0, 0, -1,
		x + 0.5, y + 0.5, z, b[4][0], b[4][1], 0, 0, -1,
		x + 0.5, y - 0.5, z, b[5][0], b[5][1], 0, 0, -1,
	}...)
	return vertices
}
