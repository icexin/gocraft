package main

import _ "embed"

var (
	//go:embed block.vert
	blockVertexSource string

	//go:embed block.frag
	blockFragmentSource string

	//go:embed line.vert
	lineVertexSource string

	//go:embed line.frag
	lineFragmentSource string

	//go:embed player.vert
	playerVertexSource string

	//go:embed player.frag
	playerFragmentSource string
)
