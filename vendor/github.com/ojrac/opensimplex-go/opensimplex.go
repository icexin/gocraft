// opensimplex is a Go implementation of Kurt Spencer's patent-free alternative
// to Perlin and Simplex noise.
//
// Given a seed, it generates smoothly-changing deterministic random values in
// 2, 3 or 4 dimensions. It's commonly used for procedurally generated images,
// geometry, or other randomly-influenced applications that require a random
// gradient.
//
// For more information on OpenSimplex noise, read more from the creator of the
// algorithm: http://uniblock.tumblr.com/post/97868843242/noise
package opensimplex

import (
	"math"
)

/**
 * OpenSimplex Noise in Go.
 * algorithm by Kurt Spencer
 * ported by Owen Raccuglia
 *
 * Based on Java v1.1 (October 5, 2014)
 */

const (
	stretchConstant2D = -0.211324865405187 // (1/Math.sqrt(2+1)-1)/2
	squishConstant2D  = 0.366025403784439  // (Math.sqrt(2+1)-1)/2
	stretchConstant3D = -1.0 / 6           // (1/Math.sqrt(3+1)-1)/3
	squishConstant3D  = 1.0 / 3            // (Math.sqrt(3+1)-1)/3
	stretchConstant4D = -0.138196601125011 // (1/Math.sqrt(4+1)-1)/4
	squishConstant4D  = 0.309016994374947  // (Math.sqrt(4+1)-1)/4

	normConstant2D = 47
	normConstant3D = 103
	normConstant4D = 30

	defaultSeed = 0
)

// A seeded Noise instance. Reusing a Noise instance (rather than recreating it
// from a known seed) will save some calculation time.
type Noise struct {
	perm            []int16
	permGradIndex3D []int16
}

// Returns a Noise instance with a seed of 0.
func New() *Noise {
	return NewWithSeed(defaultSeed)
}

// Returns a Noise instance with a 64-bit seed. Two Noise instances with the
// same seed will have the same output.
func NewWithSeed(seed int64) *Noise {
	s := Noise{
		perm:            make([]int16, 256),
		permGradIndex3D: make([]int16, 256),
	}

	source := make([]int16, 256)
	for i := range source {
		source[i] = int16(i)
	}

	seed = seed*6364136223846793005 + 1442695040888963407
	seed = seed*6364136223846793005 + 1442695040888963407
	seed = seed*6364136223846793005 + 1442695040888963407
	for i := int32(255); i >= 0; i-- {
		seed = seed*6364136223846793005 + 1442695040888963407
		r := int32((seed + 31) % int64(i+1))
		if r < 0 {
			r += i + 1
		}

		s.perm[i] = source[r]
		s.permGradIndex3D[i] = (s.perm[i] % (int16(len(gradients3D)) / 3)) * 3
		source[r] = source[i]
	}

	return &s
}

// Returns a Noise instance with a specific internal permutation state.
// If you're not sure about this, you probably want NewWithSeed().
func NewWithPerm(perm []int16) *Noise {
	s := Noise{
		perm:            perm,
		permGradIndex3D: make([]int16, 256),
	}

	for i, p := range perm {
		// Since 3D has 24 gradients, simple bitmask won't work, so precompute modulo array.
		s.permGradIndex3D[i] = (p % (int16(len(gradients3D)) / 3)) % 3
	}

	return &s
}

// Returns a random noise value in two dimensions. Repeated calls with the same
// x/y inputs will have the same output.
func (s *Noise) Eval2(x, y float64) float64 {
	// Place input coordinates onto grid.
	stretchOffset := (x + y) * stretchConstant2D
	xs := float64(x + stretchOffset)
	ys := float64(y + stretchOffset)

	// Floor to get grid coordinates of rhombus (stretched square) super-cell origin.
	xsb := int32(math.Floor(xs))
	ysb := int32(math.Floor(ys))

	// Skew out to get actual coordinates of rhombus origin. We'll need these later.
	squishOffset := float64(xsb+ysb) * squishConstant2D
	xb := float64(xsb) + squishOffset
	yb := float64(ysb) + squishOffset

	// Compute grid coordinates relative to rhombus origin.
	xins := xs - float64(xsb)
	yins := ys - float64(ysb)

	// Sum those together to get a value that determines which region we're in.
	inSum := xins + yins

	// Positions relative to origin point.
	dx0 := x - xb
	dy0 := y - yb

	// We'll be defining these inside the next block and using them afterwards.
	var dx_ext, dy_ext float64
	var xsv_ext, ysv_ext int32

	value := float64(0)

	// Contribution (1,0)
	dx1 := dx0 - 1 - squishConstant2D
	dy1 := dy0 - 0 - squishConstant2D
	attn1 := 2 - dx1*dx1 - dy1*dy1
	if attn1 > 0 {
		attn1 *= attn1
		value += attn1 * attn1 * s.extrapolate2(xsb+1, ysb+0, dx1, dy1)
	}

	// Contribution (0,1)
	dx2 := dx0 - 0 - squishConstant2D
	dy2 := dy0 - 1 - squishConstant2D
	attn2 := 2 - dx2*dx2 - dy2*dy2
	if attn2 > 0 {
		attn2 *= attn2
		value += attn2 * attn2 * s.extrapolate2(xsb+0, ysb+1, dx2, dy2)
	}

	if inSum <= 1 { // We're inside the triangle (2-Simplex) at (0,0)
		zins := 1 - inSum
		if zins > xins || zins > yins { // (0,0) is one of the closest two triangular vertices
			if xins > yins {
				xsv_ext = xsb + 1
				ysv_ext = ysb - 1
				dx_ext = dx0 - 1
				dy_ext = dy0 + 1
			} else {
				xsv_ext = xsb - 1
				ysv_ext = ysb + 1
				dx_ext = dx0 + 1
				dy_ext = dy0 - 1
			}
		} else { // (1,0) and (0,1) are the closest two vertices.
			xsv_ext = xsb + 1
			ysv_ext = ysb + 1
			dx_ext = dx0 - 1 - 2*squishConstant2D
			dy_ext = dy0 - 1 - 2*squishConstant2D
		}
	} else { // We're inside the triangle (2-Simplex) at (1,1)
		zins := 2 - inSum
		if zins < xins || zins < yins { // (0,0) is one of the closest two triangular vertices
			if xins > yins {
				xsv_ext = xsb + 2
				ysv_ext = ysb + 0
				dx_ext = dx0 - 2 - 2*squishConstant2D
				dy_ext = dy0 + 0 - 2*squishConstant2D
			} else {
				xsv_ext = xsb + 0
				ysv_ext = ysb + 2
				dx_ext = dx0 + 0 - 2*squishConstant2D
				dy_ext = dy0 - 2 - 2*squishConstant2D
			}
		} else { // (1,0) and (0,1) are the closest two vertices.
			dx_ext = dx0
			dy_ext = dy0
			xsv_ext = xsb
			ysv_ext = ysb
		}
		xsb += 1
		ysb += 1
		dx0 = dx0 - 1 - 2*squishConstant2D
		dy0 = dy0 - 1 - 2*squishConstant2D
	}

	// Contribution (0,0) or (1,1)
	attn0 := 2 - dx0*dx0 - dy0*dy0
	if attn0 > 0 {
		attn0 *= attn0
		value += attn0 * attn0 * s.extrapolate2(xsb, ysb, dx0, dy0)
	}

	// Extra Vertex
	attn_ext := 2 - dx_ext*dx_ext - dy_ext*dy_ext
	if attn_ext > 0 {
		attn_ext *= attn_ext
		value += attn_ext * attn_ext * s.extrapolate2(xsv_ext, ysv_ext, dx_ext, dy_ext)
	}

	return value / normConstant2D
}

// Returns a random noise value in three dimensions.
func (s *Noise) Eval3(x, y, z float64) float64 {
	// Place input coordinates on simplectic honeycomb.
	stretchOffset := (x + y + z) * stretchConstant3D
	xs := float64(x + stretchOffset)
	ys := float64(y + stretchOffset)
	zs := float64(z + stretchOffset)

	// Floor to get simplectic honeycomb coordinates of rhombohedron (stretched cube) super-cell origin.
	xsb := int32(math.Floor(xs))
	ysb := int32(math.Floor(ys))
	zsb := int32(math.Floor(zs))

	// Skew out to get actual coordinates of rhombohedron origin. We'll need these later.
	squishOffset := float64(xsb+ysb+zsb) * squishConstant3D
	xb := float64(xsb) + squishOffset
	yb := float64(ysb) + squishOffset
	zb := float64(zsb) + squishOffset

	// Compute simplectic honeycomb coordinates relative to rhombohedral origin.
	xins := xs - float64(xsb)
	yins := ys - float64(ysb)
	zins := zs - float64(zsb)

	// Sum those together to get a value that determines which region we're in.
	inSum := xins + yins + zins

	// Positions relative to origin point.
	dx0 := x - xb
	dy0 := y - yb
	dz0 := z - zb

	// We'll be defining these inside the next block and using them afterwards.
	var dx_ext0, dy_ext0, dz_ext0 float64
	var dx_ext1, dy_ext1, dz_ext1 float64
	var xsv_ext0, ysv_ext0, zsv_ext0 int32
	var xsv_ext1, ysv_ext1, zsv_ext1 int32

	value := float64(0)
	if inSum <= 1 { // We're inside the tetrahedron (3-Simplex) at (0,0,0)

		// Determine which two of (0,0,1), (0,1,0), (1,0,0) are closest.
		aPoint := byte(0x01)
		bPoint := byte(0x02)
		aScore := xins
		bScore := yins
		if aScore >= bScore && zins > bScore {
			bScore = zins
			bPoint = 0x04
		} else if aScore < bScore && zins > aScore {
			aScore = zins
			aPoint = 0x04
		}

		// Now we determine the two lattice points not part of the tetrahedron that may contribute.
		// This depends on the closest two tetrahedral vertices, including (0,0,0)
		wins := 1 - inSum
		if wins > aScore || wins > bScore { // (0,0,0) is one of the closest two tetrahedral vertices.
			var c byte // Our other closest vertex is the closest out of a and b.
			if bScore > aScore {
				c = bPoint
			} else {
				c = aPoint
			}

			if (c & 0x01) == 0 {
				xsv_ext0 = xsb - 1
				xsv_ext1 = xsb
				dx_ext0 = dx0 + 1
				dx_ext1 = dx0
			} else {
				xsv_ext1 = xsb + 1
				xsv_ext0 = xsv_ext1
				dx_ext1 = dx0 - 1
				dx_ext0 = dx_ext1
			}

			if (c & 0x02) == 0 {
				ysv_ext1 = ysb
				ysv_ext0 = ysv_ext1
				dy_ext1 = dy0
				dy_ext0 = dy_ext1
				if (c & 0x01) == 0 {
					ysv_ext1 -= 1
					dy_ext1 += 1
				} else {
					ysv_ext0 -= 1
					dy_ext0 += 1
				}
			} else {
				ysv_ext1 = ysb + 1
				ysv_ext0 = ysv_ext1
				dy_ext1 = dy0 - 1
				dy_ext0 = dy_ext1
			}

			if (c & 0x04) == 0 {
				zsv_ext0 = zsb
				zsv_ext1 = zsb - 1
				dz_ext0 = dz0
				dz_ext1 = dz0 + 1
			} else {
				zsv_ext1 = zsb + 1
				zsv_ext0 = zsv_ext1
				dz_ext1 = dz0 - 1
				dz_ext0 = dz_ext1
			}
		} else { // (0,0,0) is not one of the closest two tetrahedral vertices.
			c := aPoint | bPoint // Our two extra vertices are determined by the closest two.

			if (c & 0x01) == 0 {
				xsv_ext0 = xsb
				xsv_ext1 = xsb - 1
				dx_ext0 = dx0 - 2*squishConstant3D
				dx_ext1 = dx0 + 1 - squishConstant3D
			} else {
				xsv_ext1 = xsb + 1
				xsv_ext0 = xsv_ext1
				dx_ext0 = dx0 - 1 - 2*squishConstant3D
				dx_ext1 = dx0 - 1 - squishConstant3D
			}

			if (c & 0x02) == 0 {
				ysv_ext0 = ysb
				ysv_ext1 = ysb - 1
				dy_ext0 = dy0 - 2*squishConstant3D
				dy_ext1 = dy0 + 1 - squishConstant3D
			} else {
				ysv_ext1 = ysb + 1
				ysv_ext0 = ysv_ext1
				dy_ext0 = dy0 - 1 - 2*squishConstant3D
				dy_ext1 = dy0 - 1 - squishConstant3D
			}

			if (c & 0x04) == 0 {
				zsv_ext0 = zsb
				zsv_ext1 = zsb - 1
				dz_ext0 = dz0 - 2*squishConstant3D
				dz_ext1 = dz0 + 1 - squishConstant3D
			} else {
				zsv_ext1 = zsb + 1
				zsv_ext0 = zsv_ext1
				dz_ext0 = dz0 - 1 - 2*squishConstant3D
				dz_ext1 = dz0 - 1 - squishConstant3D
			}
		}

		// Contribution (0,0,0)
		attn0 := 2 - dx0*dx0 - dy0*dy0 - dz0*dz0
		if attn0 > 0 {
			attn0 *= attn0
			value += attn0 * attn0 * s.extrapolate3(xsb+0, ysb+0, zsb+0, dx0, dy0, dz0)
		}

		// Contribution (1,0,0)
		dx1 := dx0 - 1 - squishConstant3D
		dy1 := dy0 - 0 - squishConstant3D
		dz1 := dz0 - 0 - squishConstant3D
		attn1 := 2 - dx1*dx1 - dy1*dy1 - dz1*dz1
		if attn1 > 0 {
			attn1 *= attn1
			value += attn1 * attn1 * s.extrapolate3(xsb+1, ysb+0, zsb+0, dx1, dy1, dz1)
		}

		// Contribution (0,1,0)
		dx2 := dx0 - 0 - squishConstant3D
		dy2 := dy0 - 1 - squishConstant3D
		dz2 := dz1
		attn2 := 2 - dx2*dx2 - dy2*dy2 - dz2*dz2
		if attn2 > 0 {
			attn2 *= attn2
			value += attn2 * attn2 * s.extrapolate3(xsb+0, ysb+1, zsb+0, dx2, dy2, dz2)
		}

		// Contribution (0,0,1)
		dx3 := dx2
		dy3 := dy1
		dz3 := dz0 - 1 - squishConstant3D
		attn3 := 2 - dx3*dx3 - dy3*dy3 - dz3*dz3
		if attn3 > 0 {
			attn3 *= attn3
			value += attn3 * attn3 * s.extrapolate3(xsb+0, ysb+0, zsb+1, dx3, dy3, dz3)
		}
	} else if inSum >= 2 { // We're inside the tetrahedron (3-Simplex) at (1,1,1)

		// Determine which two tetrahedral vertices are the closest, out of (1,1,0), (1,0,1), (0,1,1) but not (1,1,1).
		aPoint := byte(0x06)
		aScore := xins
		bPoint := byte(0x05)
		bScore := yins
		if aScore <= bScore && zins < bScore {
			bScore = zins
			bPoint = 0x03
		} else if aScore > bScore && zins < aScore {
			aScore = zins
			aPoint = 0x03
		}

		// Now we determine the two lattice points not part of the tetrahedron that may contribute.
		// This depends on the closest two tetrahedral vertices, including (1,1,1)
		wins := 3 - inSum
		if wins < aScore || wins < bScore { // (1,1,1) is one of the closest two tetrahedral vertices.
			var c byte // Our other closest vertex is the closest out of a and b.
			if bScore < aScore {
				c = bPoint
			} else {
				c = aPoint
			}

			if (c & 0x01) != 0 {
				xsv_ext0 = xsb + 2
				xsv_ext1 = xsb + 1
				dx_ext0 = dx0 - 2 - 3*squishConstant3D
				dx_ext1 = dx0 - 1 - 3*squishConstant3D
			} else {
				xsv_ext1 = xsb
				xsv_ext0 = xsv_ext1
				dx_ext1 = dx0 - 3*squishConstant3D
				dx_ext0 = dx_ext1
			}

			if (c & 0x02) != 0 {
				ysv_ext1 = ysb + 1
				ysv_ext0 = ysv_ext1
				dy_ext1 = dy0 - 1 - 3*squishConstant3D
				dy_ext0 = dy_ext1
				if (c & 0x01) != 0 {
					ysv_ext1 += 1
					dy_ext1 -= 1
				} else {
					ysv_ext0 += 1
					dy_ext0 -= 1
				}
			} else {
				ysv_ext1 = ysb
				ysv_ext0 = ysv_ext1
				dy_ext1 = dy0 - 3*squishConstant3D
				dy_ext0 = dy_ext1
			}

			if (c & 0x04) != 0 {
				zsv_ext0 = zsb + 1
				zsv_ext1 = zsb + 2
				dz_ext0 = dz0 - 1 - 3*squishConstant3D
				dz_ext1 = dz0 - 2 - 3*squishConstant3D
			} else {
				zsv_ext1 = zsb
				zsv_ext0 = zsv_ext1
				dz_ext1 = dz0 - 3*squishConstant3D
				dz_ext0 = dz_ext1
			}
		} else { // (1,1,1) is not one of the closest two tetrahedral vertices.
			c := aPoint & bPoint // Our two extra vertices are determined by the closest two.

			if (c & 0x01) != 0 {
				xsv_ext0 = xsb + 1
				xsv_ext1 = xsb + 2
				dx_ext0 = dx0 - 1 - squishConstant3D
				dx_ext1 = dx0 - 2 - 2*squishConstant3D
			} else {
				xsv_ext1 = xsb
				xsv_ext0 = xsv_ext1
				dx_ext0 = dx0 - squishConstant3D
				dx_ext1 = dx0 - 2*squishConstant3D
			}

			if (c & 0x02) != 0 {
				ysv_ext0 = ysb + 1
				ysv_ext1 = ysb + 2
				dy_ext0 = dy0 - 1 - squishConstant3D
				dy_ext1 = dy0 - 2 - 2*squishConstant3D
			} else {
				ysv_ext1 = ysb
				ysv_ext0 = ysv_ext1
				dy_ext0 = dy0 - squishConstant3D
				dy_ext1 = dy0 - 2*squishConstant3D
			}

			if (c & 0x04) != 0 {
				zsv_ext0 = zsb + 1
				zsv_ext1 = zsb + 2
				dz_ext0 = dz0 - 1 - squishConstant3D
				dz_ext1 = dz0 - 2 - 2*squishConstant3D
			} else {
				zsv_ext1 = zsb
				zsv_ext0 = zsv_ext1
				dz_ext0 = dz0 - squishConstant3D
				dz_ext1 = dz0 - 2*squishConstant3D
			}
		}

		// Contribution (1,1,0)
		dx3 := dx0 - 1 - 2*squishConstant3D
		dy3 := dy0 - 1 - 2*squishConstant3D
		dz3 := dz0 - 0 - 2*squishConstant3D
		attn3 := 2 - dx3*dx3 - dy3*dy3 - dz3*dz3
		if attn3 > 0 {
			attn3 *= attn3
			value += attn3 * attn3 * s.extrapolate3(xsb+1, ysb+1, zsb+0, dx3, dy3, dz3)
		}

		// Contribution (1,0,1)
		dx2 := dx3
		dy2 := dy0 - 0 - 2*squishConstant3D
		dz2 := dz0 - 1 - 2*squishConstant3D
		attn2 := 2 - dx2*dx2 - dy2*dy2 - dz2*dz2
		if attn2 > 0 {
			attn2 *= attn2
			value += attn2 * attn2 * s.extrapolate3(xsb+1, ysb+0, zsb+1, dx2, dy2, dz2)
		}

		// Contribution (0,1,1)
		dx1 := dx0 - 0 - 2*squishConstant3D
		dy1 := dy3
		dz1 := dz2
		attn1 := 2 - dx1*dx1 - dy1*dy1 - dz1*dz1
		if attn1 > 0 {
			attn1 *= attn1
			value += attn1 * attn1 * s.extrapolate3(xsb+0, ysb+1, zsb+1, dx1, dy1, dz1)
		}

		// Contribution (1,1,1)
		dx0 = dx0 - 1 - 3*squishConstant3D
		dy0 = dy0 - 1 - 3*squishConstant3D
		dz0 = dz0 - 1 - 3*squishConstant3D
		attn0 := 2 - dx0*dx0 - dy0*dy0 - dz0*dz0
		if attn0 > 0 {
			attn0 *= attn0
			value += attn0 * attn0 * s.extrapolate3(xsb+1, ysb+1, zsb+1, dx0, dy0, dz0)
		}
	} else { // We're inside the octahedron (Rectified 3-Simplex) in between.
		var aScore, bScore float64
		var aPoint, bPoint byte
		var aIsFurtherSide, bIsFurtherSide bool

		// Decide between point (0,0,1) and (1,1,0) as closest
		p1 := xins + yins
		if p1 > 1 {
			aScore = p1 - 1
			aPoint = 0x03
			aIsFurtherSide = true
		} else {
			aScore = 1 - p1
			aPoint = 0x04
			aIsFurtherSide = false
		}

		// Decide between point (0,1,0) and (1,0,1) as closest
		p2 := xins + zins
		if p2 > 1 {
			bScore = p2 - 1
			bPoint = 0x05
			bIsFurtherSide = true
		} else {
			bScore = 1 - p2
			bPoint = 0x02
			bIsFurtherSide = false
		}

		// The closest out of the two (1,0,0) and (0,1,1) will replace the furthest out of the two decided above, if closer.
		p3 := yins + zins
		if p3 > 1 {
			score := p3 - 1
			if aScore <= bScore && aScore < score {
				aScore = score
				aPoint = 0x06
				aIsFurtherSide = true
			} else if aScore > bScore && bScore < score {
				bScore = score
				bPoint = 0x06
				bIsFurtherSide = true
			}
		} else {
			score := 1 - p3
			if aScore <= bScore && aScore < score {
				aScore = score
				aPoint = 0x01
				aIsFurtherSide = false
			} else if aScore > bScore && bScore < score {
				bScore = score
				bPoint = 0x01
				bIsFurtherSide = false
			}
		}

		// Where each of the two closest points are determines how the extra two vertices are calculated.
		if aIsFurtherSide == bIsFurtherSide {
			if aIsFurtherSide { // Both closest points on (1,1,1) side

				// One of the two extra points is (1,1,1)
				dx_ext0 = dx0 - 1 - 3*squishConstant3D
				dy_ext0 = dy0 - 1 - 3*squishConstant3D
				dz_ext0 = dz0 - 1 - 3*squishConstant3D
				xsv_ext0 = xsb + 1
				ysv_ext0 = ysb + 1
				zsv_ext0 = zsb + 1

				// Other extra point is based on the shared axis.
				c := aPoint & bPoint
				if (c & 0x01) != 0 {
					dx_ext1 = dx0 - 2 - 2*squishConstant3D
					dy_ext1 = dy0 - 2*squishConstant3D
					dz_ext1 = dz0 - 2*squishConstant3D
					xsv_ext1 = xsb + 2
					ysv_ext1 = ysb
					zsv_ext1 = zsb
				} else if (c & 0x02) != 0 {
					dx_ext1 = dx0 - 2*squishConstant3D
					dy_ext1 = dy0 - 2 - 2*squishConstant3D
					dz_ext1 = dz0 - 2*squishConstant3D
					xsv_ext1 = xsb
					ysv_ext1 = ysb + 2
					zsv_ext1 = zsb
				} else {
					dx_ext1 = dx0 - 2*squishConstant3D
					dy_ext1 = dy0 - 2*squishConstant3D
					dz_ext1 = dz0 - 2 - 2*squishConstant3D
					xsv_ext1 = xsb
					ysv_ext1 = ysb
					zsv_ext1 = zsb + 2
				}
			} else { // Both closest points on (0,0,0) side

				// One of the two extra points is (0,0,0)
				dx_ext0 = dx0
				dy_ext0 = dy0
				dz_ext0 = dz0
				xsv_ext0 = xsb
				ysv_ext0 = ysb
				zsv_ext0 = zsb

				// Other extra point is based on the omitted axis.
				c := aPoint | bPoint
				if (c & 0x01) == 0 {
					dx_ext1 = dx0 + 1 - squishConstant3D
					dy_ext1 = dy0 - 1 - squishConstant3D
					dz_ext1 = dz0 - 1 - squishConstant3D
					xsv_ext1 = xsb - 1
					ysv_ext1 = ysb + 1
					zsv_ext1 = zsb + 1
				} else if (c & 0x02) == 0 {
					dx_ext1 = dx0 - 1 - squishConstant3D
					dy_ext1 = dy0 + 1 - squishConstant3D
					dz_ext1 = dz0 - 1 - squishConstant3D
					xsv_ext1 = xsb + 1
					ysv_ext1 = ysb - 1
					zsv_ext1 = zsb + 1
				} else {
					dx_ext1 = dx0 - 1 - squishConstant3D
					dy_ext1 = dy0 - 1 - squishConstant3D
					dz_ext1 = dz0 + 1 - squishConstant3D
					xsv_ext1 = xsb + 1
					ysv_ext1 = ysb + 1
					zsv_ext1 = zsb - 1
				}
			}
		} else { // One point on (0,0,0) side, one point on (1,1,1) side
			var c1, c2 byte
			if aIsFurtherSide {
				c1 = aPoint
				c2 = bPoint
			} else {
				c1 = bPoint
				c2 = aPoint
			}

			// One contribution is a permutation of (1,1,-1)
			if (c1 & 0x01) == 0 {
				dx_ext0 = dx0 + 1 - squishConstant3D
				dy_ext0 = dy0 - 1 - squishConstant3D
				dz_ext0 = dz0 - 1 - squishConstant3D
				xsv_ext0 = xsb - 1
				ysv_ext0 = ysb + 1
				zsv_ext0 = zsb + 1
			} else if (c1 & 0x02) == 0 {
				dx_ext0 = dx0 - 1 - squishConstant3D
				dy_ext0 = dy0 + 1 - squishConstant3D
				dz_ext0 = dz0 - 1 - squishConstant3D
				xsv_ext0 = xsb + 1
				ysv_ext0 = ysb - 1
				zsv_ext0 = zsb + 1
			} else {
				dx_ext0 = dx0 - 1 - squishConstant3D
				dy_ext0 = dy0 - 1 - squishConstant3D
				dz_ext0 = dz0 + 1 - squishConstant3D
				xsv_ext0 = xsb + 1
				ysv_ext0 = ysb + 1
				zsv_ext0 = zsb - 1
			}

			// One contribution is a permutation of (0,0,2)
			dx_ext1 = dx0 - 2*squishConstant3D
			dy_ext1 = dy0 - 2*squishConstant3D
			dz_ext1 = dz0 - 2*squishConstant3D
			xsv_ext1 = xsb
			ysv_ext1 = ysb
			zsv_ext1 = zsb
			if (c2 & 0x01) != 0 {
				dx_ext1 -= 2
				xsv_ext1 += 2
			} else if (c2 & 0x02) != 0 {
				dy_ext1 -= 2
				ysv_ext1 += 2
			} else {
				dz_ext1 -= 2
				zsv_ext1 += 2
			}
		}

		// Contribution (1,0,0)
		dx1 := dx0 - 1 - squishConstant3D
		dy1 := dy0 - 0 - squishConstant3D
		dz1 := dz0 - 0 - squishConstant3D
		attn1 := 2 - dx1*dx1 - dy1*dy1 - dz1*dz1
		if attn1 > 0 {
			attn1 *= attn1
			value += attn1 * attn1 * s.extrapolate3(xsb+1, ysb+0, zsb+0, dx1, dy1, dz1)
		}

		// Contribution (0,1,0)
		dx2 := dx0 - 0 - squishConstant3D
		dy2 := dy0 - 1 - squishConstant3D
		dz2 := dz1
		attn2 := 2 - dx2*dx2 - dy2*dy2 - dz2*dz2
		if attn2 > 0 {
			attn2 *= attn2
			value += attn2 * attn2 * s.extrapolate3(xsb+0, ysb+1, zsb+0, dx2, dy2, dz2)
		}

		// Contribution (0,0,1)
		dx3 := dx2
		dy3 := dy1
		dz3 := dz0 - 1 - squishConstant3D
		attn3 := 2 - dx3*dx3 - dy3*dy3 - dz3*dz3
		if attn3 > 0 {
			attn3 *= attn3
			value += attn3 * attn3 * s.extrapolate3(xsb+0, ysb+0, zsb+1, dx3, dy3, dz3)
		}

		// Contribution (1,1,0)
		dx4 := dx0 - 1 - 2*squishConstant3D
		dy4 := dy0 - 1 - 2*squishConstant3D
		dz4 := dz0 - 0 - 2*squishConstant3D
		attn4 := 2 - dx4*dx4 - dy4*dy4 - dz4*dz4
		if attn4 > 0 {
			attn4 *= attn4
			value += attn4 * attn4 * s.extrapolate3(xsb+1, ysb+1, zsb+0, dx4, dy4, dz4)
		}

		// Contribution (1,0,1)
		dx5 := dx4
		dy5 := dy0 - 0 - 2*squishConstant3D
		dz5 := dz0 - 1 - 2*squishConstant3D
		attn5 := 2 - dx5*dx5 - dy5*dy5 - dz5*dz5
		if attn5 > 0 {
			attn5 *= attn5
			value += attn5 * attn5 * s.extrapolate3(xsb+1, ysb+0, zsb+1, dx5, dy5, dz5)
		}

		// Contribution (0,1,1)
		dx6 := dx0 - 0 - 2*squishConstant3D
		dy6 := dy4
		dz6 := dz5
		attn6 := 2 - dx6*dx6 - dy6*dy6 - dz6*dz6
		if attn6 > 0 {
			attn6 *= attn6
			value += attn6 * attn6 * s.extrapolate3(xsb+0, ysb+1, zsb+1, dx6, dy6, dz6)
		}
	}

	// First extra vertex
	attn_ext0 := 2 - dx_ext0*dx_ext0 - dy_ext0*dy_ext0 - dz_ext0*dz_ext0
	if attn_ext0 > 0 {
		attn_ext0 *= attn_ext0
		value += attn_ext0 * attn_ext0 * s.extrapolate3(xsv_ext0, ysv_ext0, zsv_ext0, dx_ext0, dy_ext0, dz_ext0)
	}

	// Second extra vertex
	attn_ext1 := 2 - dx_ext1*dx_ext1 - dy_ext1*dy_ext1 - dz_ext1*dz_ext1
	if attn_ext1 > 0 {
		attn_ext1 *= attn_ext1
		value += attn_ext1 * attn_ext1 * s.extrapolate3(xsv_ext1, ysv_ext1, zsv_ext1, dx_ext1, dy_ext1, dz_ext1)
	}

	return value / normConstant3D
}

// Returns a random noise value in four dimensions.
func (s *Noise) Eval4(x, y, z, w float64) float64 {
	// Place input coordinates on simplectic honeycomb.
	stretchOffset := (x + y + z + w) * stretchConstant4D
	xs := x + stretchOffset
	ys := y + stretchOffset
	zs := z + stretchOffset
	ws := w + stretchOffset

	// Floor to get simplectic honeycomb coordinates of rhombo-hypercube super-cell origin.
	xsb := int32(math.Floor(xs))
	ysb := int32(math.Floor(ys))
	zsb := int32(math.Floor(zs))
	wsb := int32(math.Floor(ws))

	// Skew out to get actual coordinates of stretched rhombo-hypercube origin. We'll need these later.
	squishOffset := float64(xsb+ysb+zsb+wsb) * squishConstant4D
	xb := float64(xsb) + squishOffset
	yb := float64(ysb) + squishOffset
	zb := float64(zsb) + squishOffset
	wb := float64(wsb) + squishOffset

	// Compute simplectic honeycomb coordinates relative to rhombo-hypercube origin.
	xins := xs - float64(xsb)
	yins := ys - float64(ysb)
	zins := zs - float64(zsb)
	wins := ws - float64(wsb)

	// Sum those together to get a value that determines which region we're in.
	inSum := xins + yins + zins + wins

	// Positions relative to origin point.
	dx0 := x - xb
	dy0 := y - yb
	dz0 := z - zb
	dw0 := w - wb

	// We'll be defining these inside the next block and using them afterwards.
	var dx_ext0, dy_ext0, dz_ext0, dw_ext0 float64
	var dx_ext1, dy_ext1, dz_ext1, dw_ext1 float64
	var dx_ext2, dy_ext2, dz_ext2, dw_ext2 float64
	var xsv_ext0, ysv_ext0, zsv_ext0, wsv_ext0 int32
	var xsv_ext1, ysv_ext1, zsv_ext1, wsv_ext1 int32
	var xsv_ext2, ysv_ext2, zsv_ext2, wsv_ext2 int32

	var value float64 = 0
	if inSum <= 1 { // We're inside the pentachoron (4-Simplex) at (0,0,0,0)
		// Determine which two of (0,0,0,1), (0,0,1,0), (0,1,0,0), (1,0,0,0) are closest.
		var aPoint byte = 0x01
		aScore := xins
		var bPoint byte = 0x02
		bScore := yins
		if aScore >= bScore && zins > bScore {
			bScore = zins
			bPoint = 0x04
		} else if aScore < bScore && zins > aScore {
			aScore = zins
			aPoint = 0x04
		}
		if aScore >= bScore && wins > bScore {
			bScore = wins
			bPoint = 0x08
		} else if aScore < bScore && wins > aScore {
			aScore = wins
			aPoint = 0x08
		}

		// Now we determine the three lattice points not part of the pentachoron that may contribute.
		// This depends on the closest two pentachoron vertices, including (0,0,0,0)
		uins := 1 - inSum
		if uins > aScore || uins > bScore { // (0,0,0,0) is one of the closest two pentachoron vertices.
			var c byte
			// Our other closest vertex is the closest out of a and b.
			if bScore > aScore {
				c = bPoint
			} else {
				c = aPoint
			}
			if (c & 0x01) == 0 {
				xsv_ext0 = xsb - 1
				xsv_ext2 = xsb
				xsv_ext1 = xsv_ext2
				dx_ext0 = dx0 + 1
				dx_ext2 = dx0
				dx_ext1 = dx_ext2
			} else {
				xsv_ext2 = xsb + 1
				xsv_ext1 = xsv_ext2
				xsv_ext0 = xsv_ext1
				dx_ext2 = dx0 - 1
				dx_ext1 = dx_ext2
				dx_ext0 = dx_ext1
			}

			if (c & 0x02) == 0 {
				ysv_ext2 = ysb
				ysv_ext1 = ysv_ext2
				ysv_ext0 = ysv_ext1
				dy_ext2 = dy0
				dy_ext1 = dy_ext2
				dy_ext0 = dy_ext1
				if (c & 0x01) == 0x01 {
					ysv_ext0 -= 1
					dy_ext0 += 1
				} else {
					ysv_ext1 -= 1
					dy_ext1 += 1
				}
			} else {
				ysv_ext2 = ysb + 1
				ysv_ext1 = ysv_ext2
				ysv_ext0 = ysv_ext1
				dy_ext2 = dy0 - 1
				dy_ext1 = dy_ext2
				dy_ext0 = dy_ext1
			}

			if (c & 0x04) == 0 {
				zsv_ext2 = zsb
				zsv_ext1 = zsv_ext2
				zsv_ext0 = zsv_ext1
				dz_ext2 = dz0
				dz_ext1 = dz_ext2
				dz_ext0 = dz_ext1
				if (c & 0x03) != 0 {
					if (c & 0x03) == 0x03 {
						zsv_ext0 -= 1
						dz_ext0 += 1
					} else {
						zsv_ext1 -= 1
						dz_ext1 += 1
					}
				} else {
					zsv_ext2 -= 1
					dz_ext2 += 1
				}
			} else {
				zsv_ext2 = zsb + 1
				zsv_ext1 = zsv_ext2
				zsv_ext0 = zsv_ext1
				dz_ext2 = dz0 - 1
				dz_ext1 = dz_ext2
				dz_ext0 = dz_ext1
			}

			if (c & 0x08) == 0 {
				wsv_ext1 = wsb
				wsv_ext0 = wsv_ext1
				wsv_ext2 = wsb - 1
				dw_ext1 = dw0
				dw_ext0 = dw_ext1
				dw_ext2 = dw0 + 1
			} else {
				wsv_ext2 = wsb + 1
				wsv_ext1 = wsv_ext2
				wsv_ext0 = wsv_ext1
				dw_ext2 = dw0 - 1
				dw_ext1 = dw_ext2
				dw_ext0 = dw_ext1
			}
		} else { // (0,0,0,0) is not one of the closest two pentachoron vertices.
			c := aPoint | bPoint // Our three extra vertices are determined by the closest two.

			if (c & 0x01) == 0 {
				xsv_ext2 = xsb
				xsv_ext0 = xsv_ext2
				xsv_ext1 = xsb - 1
				dx_ext0 = dx0 - 2*squishConstant4D
				dx_ext1 = dx0 + 1 - squishConstant4D
				dx_ext2 = dx0 - squishConstant4D
			} else {
				xsv_ext2 = xsb + 1
				xsv_ext1 = xsv_ext2
				xsv_ext0 = xsv_ext1
				dx_ext0 = dx0 - 1 - 2*squishConstant4D
				dx_ext2 = dx0 - 1 - squishConstant4D
				dx_ext1 = dx_ext2
			}

			if (c & 0x02) == 0 {
				ysv_ext2 = ysb
				ysv_ext1 = ysv_ext2
				ysv_ext0 = ysv_ext1
				dy_ext0 = dy0 - 2*squishConstant4D
				dy_ext2 = dy0 - squishConstant4D
				dy_ext1 = dy_ext2
				if (c & 0x01) == 0x01 {
					ysv_ext1 -= 1
					dy_ext1 += 1
				} else {
					ysv_ext2 -= 1
					dy_ext2 += 1
				}
			} else {
				ysv_ext2 = ysb + 1
				ysv_ext1 = ysv_ext2
				ysv_ext0 = ysv_ext1
				dy_ext0 = dy0 - 1 - 2*squishConstant4D
				dy_ext2 = dy0 - 1 - squishConstant4D
				dy_ext1 = dy_ext2
			}

			if (c & 0x04) == 0 {
				zsv_ext2 = zsb
				zsv_ext1 = zsv_ext2
				zsv_ext0 = zsv_ext1
				dz_ext0 = dz0 - 2*squishConstant4D
				dz_ext2 = dz0 - squishConstant4D
				dz_ext1 = dz_ext2
				if (c & 0x03) == 0x03 {
					zsv_ext1 -= 1
					dz_ext1 += 1
				} else {
					zsv_ext2 -= 1
					dz_ext2 += 1
				}
			} else {
				zsv_ext2 = zsb + 1
				zsv_ext1 = zsv_ext2
				zsv_ext0 = zsv_ext1
				dz_ext0 = dz0 - 1 - 2*squishConstant4D
				dz_ext2 = dz0 - 1 - squishConstant4D
				dz_ext1 = dz_ext2
			}

			if (c & 0x08) == 0 {
				wsv_ext1 = wsb
				wsv_ext0 = wsv_ext1
				wsv_ext2 = wsb - 1
				dw_ext0 = dw0 - 2*squishConstant4D
				dw_ext1 = dw0 - squishConstant4D
				dw_ext2 = dw0 + 1 - squishConstant4D
			} else {
				wsv_ext2 = wsb + 1
				wsv_ext1 = wsv_ext2
				wsv_ext0 = wsv_ext1
				dw_ext0 = dw0 - 1 - 2*squishConstant4D
				dw_ext2 = dw0 - 1 - squishConstant4D
				dw_ext1 = dw_ext2
			}
		}

		// Contribution (0,0,0,0)
		attn0 := 2 - dx0*dx0 - dy0*dy0 - dz0*dz0 - dw0*dw0
		if attn0 > 0 {
			attn0 *= attn0
			value += attn0 * attn0 * s.extrapolate4(xsb+0, ysb+0, zsb+0, wsb+0, dx0, dy0, dz0, dw0)
		}

		// Contribution (1,0,0,0)
		dx1 := dx0 - 1 - squishConstant4D
		dy1 := dy0 - 0 - squishConstant4D
		dz1 := dz0 - 0 - squishConstant4D
		dw1 := dw0 - 0 - squishConstant4D
		attn1 := 2 - dx1*dx1 - dy1*dy1 - dz1*dz1 - dw1*dw1
		if attn1 > 0 {
			attn1 *= attn1
			value += attn1 * attn1 * s.extrapolate4(xsb+1, ysb+0, zsb+0, wsb+0, dx1, dy1, dz1, dw1)
		}

		// Contribution (0,1,0,0)
		dx2 := dx0 - 0 - squishConstant4D
		dy2 := dy0 - 1 - squishConstant4D
		dz2 := dz1
		dw2 := dw1
		attn2 := 2 - dx2*dx2 - dy2*dy2 - dz2*dz2 - dw2*dw2
		if attn2 > 0 {
			attn2 *= attn2
			value += attn2 * attn2 * s.extrapolate4(xsb+0, ysb+1, zsb+0, wsb+0, dx2, dy2, dz2, dw2)
		}

		// Contribution (0,0,1,0)
		dx3 := dx2
		dy3 := dy1
		dz3 := dz0 - 1 - squishConstant4D
		dw3 := dw1
		attn3 := 2 - dx3*dx3 - dy3*dy3 - dz3*dz3 - dw3*dw3
		if attn3 > 0 {
			attn3 *= attn3
			value += attn3 * attn3 * s.extrapolate4(xsb+0, ysb+0, zsb+1, wsb+0, dx3, dy3, dz3, dw3)
		}

		// Contribution (0,0,0,1)
		dx4 := dx2
		dy4 := dy1
		dz4 := dz1
		dw4 := dw0 - 1 - squishConstant4D
		attn4 := 2 - dx4*dx4 - dy4*dy4 - dz4*dz4 - dw4*dw4
		if attn4 > 0 {
			attn4 *= attn4
			value += attn4 * attn4 * s.extrapolate4(xsb+0, ysb+0, zsb+0, wsb+1, dx4, dy4, dz4, dw4)
		}
	} else if inSum >= 3 { // We're inside the pentachoron (4-Simplex) at (1,1,1,1)
		// Determine which two of (1,1,1,0), (1,1,0,1), (1,0,1,1), (0,1,1,1) are closest.
		var aPoint byte = 0x0E
		aScore := xins
		var bPoint byte = 0x0D
		bScore := yins
		if aScore <= bScore && zins < bScore {
			bScore = zins
			bPoint = 0x0B
		} else if aScore > bScore && zins < aScore {
			aScore = zins
			aPoint = 0x0B
		}
		if aScore <= bScore && wins < bScore {
			bScore = wins
			bPoint = 0x07
		} else if aScore > bScore && wins < aScore {
			aScore = wins
			aPoint = 0x07
		}

		// Now we determine the three lattice points not part of the pentachoron that may contribute.
		// This depends on the closest two pentachoron vertices, including (0,0,0,0)
		uins := 4 - inSum
		if uins < aScore || uins < bScore { // (1,1,1,1) is one of the closest two pentachoron vertices.
			var c byte
			// Our other closest vertex is the closest out of a and b.
			if bScore < aScore {
				c = bPoint
			} else {
				c = aPoint
			}

			if (c & 0x01) != 0 {
				xsv_ext0 = xsb + 2
				xsv_ext2 = xsb + 1
				xsv_ext1 = xsv_ext2
				dx_ext0 = dx0 - 2 - 4*squishConstant4D
				dx_ext2 = dx0 - 1 - 4*squishConstant4D
				dx_ext1 = dx_ext2
			} else {
				xsv_ext2 = xsb
				xsv_ext1 = xsv_ext2
				xsv_ext0 = xsv_ext1
				dx_ext2 = dx0 - 4*squishConstant4D
				dx_ext1 = dx_ext2
				dx_ext0 = dx_ext1
			}

			if (c & 0x02) != 0 {
				ysv_ext2 = ysb + 1
				ysv_ext1 = ysv_ext2
				ysv_ext0 = ysv_ext1
				dy_ext2 = dy0 - 1 - 4*squishConstant4D
				dy_ext1 = dy_ext2
				dy_ext0 = dy_ext1
				if (c & 0x01) != 0 {
					ysv_ext1 += 1
					dy_ext1 -= 1
				} else {
					ysv_ext0 += 1
					dy_ext0 -= 1
				}
			} else {
				ysv_ext2 = ysb
				ysv_ext1 = ysv_ext2
				ysv_ext0 = ysv_ext1
				dy_ext2 = dy0 - 4*squishConstant4D
				dy_ext1 = dy_ext2
				dy_ext0 = dy_ext1
			}

			if (c & 0x04) != 0 {
				zsv_ext2 = zsb + 1
				zsv_ext1 = zsv_ext2
				zsv_ext0 = zsv_ext1
				dz_ext2 = dz0 - 1 - 4*squishConstant4D
				dz_ext1 = dz_ext2
				dz_ext0 = dz_ext1
				if (c & 0x03) != 0x03 {
					if (c & 0x03) == 0 {
						zsv_ext0 += 1
						dz_ext0 -= 1
					} else {
						zsv_ext1 += 1
						dz_ext1 -= 1
					}
				} else {
					zsv_ext2 += 1
					dz_ext2 -= 1
				}
			} else {
				zsv_ext2 = zsb
				zsv_ext1 = zsv_ext2
				zsv_ext0 = zsv_ext1
				dz_ext2 = dz0 - 4*squishConstant4D
				dz_ext1 = dz_ext2
				dz_ext0 = dz_ext1
			}

			if (c & 0x08) != 0 {
				wsv_ext1 = wsb + 1
				wsv_ext0 = wsv_ext1
				wsv_ext2 = wsb + 2
				dw_ext1 = dw0 - 1 - 4*squishConstant4D
				dw_ext0 = dw_ext1
				dw_ext2 = dw0 - 2 - 4*squishConstant4D
			} else {
				wsv_ext2 = wsb
				wsv_ext1 = wsv_ext2
				wsv_ext0 = wsv_ext1
				dw_ext2 = dw0 - 4*squishConstant4D
				dw_ext1 = dw_ext2
				dw_ext0 = dw_ext1
			}
		} else { // (1,1,1,1) is not one of the closest two pentachoron vertices.
			c := aPoint & bPoint // Our three extra vertices are determined by the closest two.

			if (c & 0x01) != 0 {
				xsv_ext2 = xsb + 1
				xsv_ext0 = xsv_ext2
				xsv_ext1 = xsb + 2
				dx_ext0 = dx0 - 1 - 2*squishConstant4D
				dx_ext1 = dx0 - 2 - 3*squishConstant4D
				dx_ext2 = dx0 - 1 - 3*squishConstant4D
			} else {
				xsv_ext2 = xsb
				xsv_ext1 = xsv_ext2
				xsv_ext0 = xsv_ext1
				dx_ext0 = dx0 - 2*squishConstant4D
				dx_ext2 = dx0 - 3*squishConstant4D
				dx_ext1 = dx_ext2
			}

			if (c & 0x02) != 0 {
				ysv_ext2 = ysb + 1
				ysv_ext1 = ysv_ext2
				ysv_ext0 = ysv_ext1
				dy_ext0 = dy0 - 1 - 2*squishConstant4D
				dy_ext2 = dy0 - 1 - 3*squishConstant4D
				dy_ext1 = dy_ext2
				if (c & 0x01) != 0 {
					ysv_ext2 += 1
					dy_ext2 -= 1
				} else {
					ysv_ext1 += 1
					dy_ext1 -= 1
				}
			} else {
				ysv_ext2 = ysb
				ysv_ext1 = ysv_ext2
				ysv_ext0 = ysv_ext1
				dy_ext0 = dy0 - 2*squishConstant4D
				dy_ext2 = dy0 - 3*squishConstant4D
				dy_ext1 = dy_ext2
			}

			if (c & 0x04) != 0 {
				zsv_ext2 = zsb + 1
				zsv_ext1 = zsv_ext2
				zsv_ext0 = zsv_ext1
				dz_ext0 = dz0 - 1 - 2*squishConstant4D
				dz_ext2 = dz0 - 1 - 3*squishConstant4D
				dz_ext1 = dz_ext2
				if (c & 0x03) != 0 {
					zsv_ext2 += 1
					dz_ext2 -= 1
				} else {
					zsv_ext1 += 1
					dz_ext1 -= 1
				}
			} else {
				zsv_ext2 = zsb
				zsv_ext1 = zsv_ext2
				zsv_ext0 = zsv_ext1
				dz_ext0 = dz0 - 2*squishConstant4D
				dz_ext2 = dz0 - 3*squishConstant4D
				dz_ext1 = dz_ext2
			}

			if (c & 0x08) != 0 {
				wsv_ext1 = wsb + 1
				wsv_ext0 = wsv_ext1
				wsv_ext2 = wsb + 2
				dw_ext0 = dw0 - 1 - 2*squishConstant4D
				dw_ext1 = dw0 - 1 - 3*squishConstant4D
				dw_ext2 = dw0 - 2 - 3*squishConstant4D
			} else {
				wsv_ext2 = wsb
				wsv_ext1 = wsv_ext2
				wsv_ext0 = wsv_ext1
				dw_ext0 = dw0 - 2*squishConstant4D
				dw_ext2 = dw0 - 3*squishConstant4D
				dw_ext1 = dw_ext2
			}
		}

		// Contribution (1,1,1,0)
		dx4 := dx0 - 1 - 3*squishConstant4D
		dy4 := dy0 - 1 - 3*squishConstant4D
		dz4 := dz0 - 1 - 3*squishConstant4D
		dw4 := dw0 - 3*squishConstant4D
		attn4 := 2 - dx4*dx4 - dy4*dy4 - dz4*dz4 - dw4*dw4
		if attn4 > 0 {
			attn4 *= attn4
			value += attn4 * attn4 * s.extrapolate4(xsb+1, ysb+1, zsb+1, wsb+0, dx4, dy4, dz4, dw4)
		}

		// Contribution (1,1,0,1)
		dx3 := dx4
		dy3 := dy4
		dz3 := dz0 - 3*squishConstant4D
		dw3 := dw0 - 1 - 3*squishConstant4D
		attn3 := 2 - dx3*dx3 - dy3*dy3 - dz3*dz3 - dw3*dw3
		if attn3 > 0 {
			attn3 *= attn3
			value += attn3 * attn3 * s.extrapolate4(xsb+1, ysb+1, zsb+0, wsb+1, dx3, dy3, dz3, dw3)
		}

		// Contribution (1,0,1,1)
		dx2 := dx4
		dy2 := dy0 - 3*squishConstant4D
		dz2 := dz4
		dw2 := dw3
		attn2 := 2 - dx2*dx2 - dy2*dy2 - dz2*dz2 - dw2*dw2
		if attn2 > 0 {
			attn2 *= attn2
			value += attn2 * attn2 * s.extrapolate4(xsb+1, ysb+0, zsb+1, wsb+1, dx2, dy2, dz2, dw2)
		}

		// Contribution (0,1,1,1)
		dx1 := dx0 - 3*squishConstant4D
		dz1 := dz4
		dy1 := dy4
		dw1 := dw3
		attn1 := 2 - dx1*dx1 - dy1*dy1 - dz1*dz1 - dw1*dw1
		if attn1 > 0 {
			attn1 *= attn1
			value += attn1 * attn1 * s.extrapolate4(xsb+0, ysb+1, zsb+1, wsb+1, dx1, dy1, dz1, dw1)
		}

		// Contribution (1,1,1,1)
		dx0 = dx0 - 1 - 4*squishConstant4D
		dy0 = dy0 - 1 - 4*squishConstant4D
		dz0 = dz0 - 1 - 4*squishConstant4D
		dw0 = dw0 - 1 - 4*squishConstant4D
		attn0 := 2 - dx0*dx0 - dy0*dy0 - dz0*dz0 - dw0*dw0
		if attn0 > 0 {
			attn0 *= attn0
			value += attn0 * attn0 * s.extrapolate4(xsb+1, ysb+1, zsb+1, wsb+1, dx0, dy0, dz0, dw0)
		}
	} else if inSum <= 2 { // We're inside the first dispentachoron (Rectified 4-Simplex)
		var aScore, bScore float64
		var aPoint, bPoint byte
		var aIsBiggerSide bool = true
		var bIsBiggerSide bool = true

		// Decide between (1,1,0,0) and (0,0,1,1)
		if xins+yins > zins+wins {
			aScore = xins + yins
			aPoint = 0x03
		} else {
			aScore = zins + wins
			aPoint = 0x0C
		}

		// Decide between (1,0,1,0) and (0,1,0,1)
		if xins+zins > yins+wins {
			bScore = xins + zins
			bPoint = 0x05
		} else {
			bScore = yins + wins
			bPoint = 0x0A
		}

		// Closer between (1,0,0,1) and (0,1,1,0) will replace the further of a and b, if closer.
		if xins+wins > yins+zins {
			score := xins + wins
			if aScore >= bScore && score > bScore {
				bScore = score
				bPoint = 0x09
			} else if aScore < bScore && score > aScore {
				aScore = score
				aPoint = 0x09
			}
		} else {
			score := yins + zins
			if aScore >= bScore && score > bScore {
				bScore = score
				bPoint = 0x06
			} else if aScore < bScore && score > aScore {
				aScore = score
				aPoint = 0x06
			}
		}

		// Decide if (1,0,0,0) is closer.
		p1 := 2 - inSum + xins
		if aScore >= bScore && p1 > bScore {
			bScore = p1
			bPoint = 0x01
			bIsBiggerSide = false
		} else if aScore < bScore && p1 > aScore {
			aScore = p1
			aPoint = 0x01
			aIsBiggerSide = false
		}

		// Decide if (0,1,0,0) is closer.
		p2 := 2 - inSum + yins
		if aScore >= bScore && p2 > bScore {
			bScore = p2
			bPoint = 0x02
			bIsBiggerSide = false
		} else if aScore < bScore && p2 > aScore {
			aScore = p2
			aPoint = 0x02
			aIsBiggerSide = false
		}

		// Decide if (0,0,1,0) is closer.
		p3 := 2 - inSum + zins
		if aScore >= bScore && p3 > bScore {
			bScore = p3
			bPoint = 0x04
			bIsBiggerSide = false
		} else if aScore < bScore && p3 > aScore {
			aScore = p3
			aPoint = 0x04
			aIsBiggerSide = false
		}

		// Decide if (0,0,0,1) is closer.
		p4 := 2 - inSum + wins
		if aScore >= bScore && p4 > bScore {
			bScore = p4
			bPoint = 0x08
			bIsBiggerSide = false
		} else if aScore < bScore && p4 > aScore {
			aScore = p4
			aPoint = 0x08
			aIsBiggerSide = false
		}

		// Where each of the two closest points are determines how the extra three vertices are calculated.
		if aIsBiggerSide == bIsBiggerSide {
			if aIsBiggerSide { // Both closest points on the bigger side
				c1 := aPoint | bPoint
				c2 := aPoint & bPoint
				if (c1 & 0x01) == 0 {
					xsv_ext0 = xsb
					xsv_ext1 = xsb - 1
					dx_ext0 = dx0 - 3*squishConstant4D
					dx_ext1 = dx0 + 1 - 2*squishConstant4D
				} else {
					xsv_ext1 = xsb + 1
					xsv_ext0 = xsv_ext1
					dx_ext0 = dx0 - 1 - 3*squishConstant4D
					dx_ext1 = dx0 - 1 - 2*squishConstant4D
				}

				if (c1 & 0x02) == 0 {
					ysv_ext0 = ysb
					ysv_ext1 = ysb - 1
					dy_ext0 = dy0 - 3*squishConstant4D
					dy_ext1 = dy0 + 1 - 2*squishConstant4D
				} else {
					ysv_ext1 = ysb + 1
					ysv_ext0 = ysv_ext1
					dy_ext0 = dy0 - 1 - 3*squishConstant4D
					dy_ext1 = dy0 - 1 - 2*squishConstant4D
				}

				if (c1 & 0x04) == 0 {
					zsv_ext0 = zsb
					zsv_ext1 = zsb - 1
					dz_ext0 = dz0 - 3*squishConstant4D
					dz_ext1 = dz0 + 1 - 2*squishConstant4D
				} else {
					zsv_ext1 = zsb + 1
					zsv_ext0 = zsv_ext1
					dz_ext0 = dz0 - 1 - 3*squishConstant4D
					dz_ext1 = dz0 - 1 - 2*squishConstant4D
				}

				if (c1 & 0x08) == 0 {
					wsv_ext0 = wsb
					wsv_ext1 = wsb - 1
					dw_ext0 = dw0 - 3*squishConstant4D
					dw_ext1 = dw0 + 1 - 2*squishConstant4D
				} else {
					wsv_ext1 = wsb + 1
					wsv_ext0 = wsv_ext1
					dw_ext0 = dw0 - 1 - 3*squishConstant4D
					dw_ext1 = dw0 - 1 - 2*squishConstant4D
				}

				// One combination is a permutation of (0,0,0,2) based on c2
				xsv_ext2 = xsb
				ysv_ext2 = ysb
				zsv_ext2 = zsb
				wsv_ext2 = wsb
				dx_ext2 = dx0 - 2*squishConstant4D
				dy_ext2 = dy0 - 2*squishConstant4D
				dz_ext2 = dz0 - 2*squishConstant4D
				dw_ext2 = dw0 - 2*squishConstant4D
				if (c2 & 0x01) != 0 {
					xsv_ext2 += 2
					dx_ext2 -= 2
				} else if (c2 & 0x02) != 0 {
					ysv_ext2 += 2
					dy_ext2 -= 2
				} else if (c2 & 0x04) != 0 {
					zsv_ext2 += 2
					dz_ext2 -= 2
				} else {
					wsv_ext2 += 2
					dw_ext2 -= 2
				}

			} else { // Both closest points on the smaller side
				// One of the two extra points is (0,0,0,0)
				xsv_ext2 = xsb
				ysv_ext2 = ysb
				zsv_ext2 = zsb
				wsv_ext2 = wsb
				dx_ext2 = dx0
				dy_ext2 = dy0
				dz_ext2 = dz0
				dw_ext2 = dw0

				// Other two points are based on the omitted axes.
				c := aPoint | bPoint

				if (c & 0x01) == 0 {
					xsv_ext0 = xsb - 1
					xsv_ext1 = xsb
					dx_ext0 = dx0 + 1 - squishConstant4D
					dx_ext1 = dx0 - squishConstant4D
				} else {
					xsv_ext1 = xsb + 1
					xsv_ext0 = xsv_ext1
					dx_ext1 = dx0 - 1 - squishConstant4D
					dx_ext0 = dx_ext1
				}

				if (c & 0x02) == 0 {
					ysv_ext1 = ysb
					ysv_ext0 = ysv_ext1
					dy_ext1 = dy0 - squishConstant4D
					dy_ext0 = dy_ext1
					if (c & 0x01) == 0x01 {
						ysv_ext0 -= 1
						dy_ext0 += 1
					} else {
						ysv_ext1 -= 1
						dy_ext1 += 1
					}
				} else {
					ysv_ext1 = ysb + 1
					ysv_ext0 = ysv_ext1
					dy_ext1 = dy0 - 1 - squishConstant4D
					dy_ext0 = dy_ext1
				}

				if (c & 0x04) == 0 {
					zsv_ext1 = zsb
					zsv_ext0 = zsv_ext1
					dz_ext1 = dz0 - squishConstant4D
					dz_ext0 = dz_ext1
					if (c & 0x03) == 0x03 {
						zsv_ext0 -= 1
						dz_ext0 += 1
					} else {
						zsv_ext1 -= 1
						dz_ext1 += 1
					}
				} else {
					zsv_ext1 = zsb + 1
					zsv_ext0 = zsv_ext1
					dz_ext1 = dz0 - 1 - squishConstant4D
					dz_ext0 = dz_ext1
				}

				if (c & 0x08) == 0 {
					wsv_ext0 = wsb
					wsv_ext1 = wsb - 1
					dw_ext0 = dw0 - squishConstant4D
					dw_ext1 = dw0 + 1 - squishConstant4D
				} else {
					wsv_ext1 = wsb + 1
					wsv_ext0 = wsv_ext1
					dw_ext1 = dw0 - 1 - squishConstant4D
					dw_ext0 = dw_ext1
				}

			}
		} else { // One point on each "side"
			var c1, c2 byte
			if aIsBiggerSide {
				c1 = aPoint
				c2 = bPoint
			} else {
				c1 = bPoint
				c2 = aPoint
			}

			// Two contributions are the bigger-sided point with each 0 replaced with -1.
			if (c1 & 0x01) == 0 {
				xsv_ext0 = xsb - 1
				xsv_ext1 = xsb
				dx_ext0 = dx0 + 1 - squishConstant4D
				dx_ext1 = dx0 - squishConstant4D
			} else {
				xsv_ext1 = xsb + 1
				xsv_ext0 = xsv_ext1
				dx_ext1 = dx0 - 1 - squishConstant4D
				dx_ext0 = dx_ext1
			}

			if (c1 & 0x02) == 0 {
				ysv_ext1 = ysb
				ysv_ext0 = ysv_ext1
				dy_ext1 = dy0 - squishConstant4D
				dy_ext0 = dy_ext1
				if (c1 & 0x01) == 0x01 {
					ysv_ext0 -= 1
					dy_ext0 += 1
				} else {
					ysv_ext1 -= 1
					dy_ext1 += 1
				}
			} else {
				ysv_ext1 = ysb + 1
				ysv_ext0 = ysv_ext1
				dy_ext1 = dy0 - 1 - squishConstant4D
				dy_ext0 = dy_ext1
			}

			if (c1 & 0x04) == 0 {
				zsv_ext1 = zsb
				zsv_ext0 = zsv_ext1
				dz_ext1 = dz0 - squishConstant4D
				dz_ext0 = dz_ext1
				if (c1 & 0x03) == 0x03 {
					zsv_ext0 -= 1
					dz_ext0 += 1
				} else {
					zsv_ext1 -= 1
					dz_ext1 += 1
				}
			} else {
				zsv_ext1 = zsb + 1
				zsv_ext0 = zsv_ext1
				dz_ext1 = dz0 - 1 - squishConstant4D
				dz_ext0 = dz_ext1
			}

			if (c1 & 0x08) == 0 {
				wsv_ext0 = wsb
				wsv_ext1 = wsb - 1
				dw_ext0 = dw0 - squishConstant4D
				dw_ext1 = dw0 + 1 - squishConstant4D
			} else {
				wsv_ext1 = wsb + 1
				wsv_ext0 = wsv_ext1
				dw_ext1 = dw0 - 1 - squishConstant4D
				dw_ext0 = dw_ext1
			}

			// One contribution is a permutation of (0,0,0,2) based on the smaller-sided point
			xsv_ext2 = xsb
			ysv_ext2 = ysb
			zsv_ext2 = zsb
			wsv_ext2 = wsb
			dx_ext2 = dx0 - 2*squishConstant4D
			dy_ext2 = dy0 - 2*squishConstant4D
			dz_ext2 = dz0 - 2*squishConstant4D
			dw_ext2 = dw0 - 2*squishConstant4D
			if (c2 & 0x01) != 0 {
				xsv_ext2 += 2
				dx_ext2 -= 2
			} else if (c2 & 0x02) != 0 {
				ysv_ext2 += 2
				dy_ext2 -= 2
			} else if (c2 & 0x04) != 0 {
				zsv_ext2 += 2
				dz_ext2 -= 2
			} else {
				wsv_ext2 += 2
				dw_ext2 -= 2
			}
		}

		// Contribution (1,0,0,0)
		dx1 := dx0 - 1 - squishConstant4D
		dy1 := dy0 - 0 - squishConstant4D
		dz1 := dz0 - 0 - squishConstant4D
		dw1 := dw0 - 0 - squishConstant4D
		attn1 := 2 - dx1*dx1 - dy1*dy1 - dz1*dz1 - dw1*dw1
		if attn1 > 0 {
			attn1 *= attn1
			value += attn1 * attn1 * s.extrapolate4(xsb+1, ysb+0, zsb+0, wsb+0, dx1, dy1, dz1, dw1)
		}

		// Contribution (0,1,0,0)
		dx2 := dx0 - 0 - squishConstant4D
		dy2 := dy0 - 1 - squishConstant4D
		dz2 := dz1
		dw2 := dw1
		attn2 := 2 - dx2*dx2 - dy2*dy2 - dz2*dz2 - dw2*dw2
		if attn2 > 0 {
			attn2 *= attn2
			value += attn2 * attn2 * s.extrapolate4(xsb+0, ysb+1, zsb+0, wsb+0, dx2, dy2, dz2, dw2)
		}

		// Contribution (0,0,1,0)
		dx3 := dx2
		dy3 := dy1
		dz3 := dz0 - 1 - squishConstant4D
		dw3 := dw1
		attn3 := 2 - dx3*dx3 - dy3*dy3 - dz3*dz3 - dw3*dw3
		if attn3 > 0 {
			attn3 *= attn3
			value += attn3 * attn3 * s.extrapolate4(xsb+0, ysb+0, zsb+1, wsb+0, dx3, dy3, dz3, dw3)
		}

		// Contribution (0,0,0,1)
		dx4 := dx2
		dy4 := dy1
		dz4 := dz1
		dw4 := dw0 - 1 - squishConstant4D
		attn4 := 2 - dx4*dx4 - dy4*dy4 - dz4*dz4 - dw4*dw4
		if attn4 > 0 {
			attn4 *= attn4
			value += attn4 * attn4 * s.extrapolate4(xsb+0, ysb+0, zsb+0, wsb+1, dx4, dy4, dz4, dw4)
		}

		// Contribution (1,1,0,0)
		dx5 := dx0 - 1 - 2*squishConstant4D
		dy5 := dy0 - 1 - 2*squishConstant4D
		dz5 := dz0 - 0 - 2*squishConstant4D
		dw5 := dw0 - 0 - 2*squishConstant4D
		attn5 := 2 - dx5*dx5 - dy5*dy5 - dz5*dz5 - dw5*dw5
		if attn5 > 0 {
			attn5 *= attn5
			value += attn5 * attn5 * s.extrapolate4(xsb+1, ysb+1, zsb+0, wsb+0, dx5, dy5, dz5, dw5)
		}

		// Contribution (1,0,1,0)
		dx6 := dx0 - 1 - 2*squishConstant4D
		dy6 := dy0 - 0 - 2*squishConstant4D
		dz6 := dz0 - 1 - 2*squishConstant4D
		dw6 := dw0 - 0 - 2*squishConstant4D
		attn6 := 2 - dx6*dx6 - dy6*dy6 - dz6*dz6 - dw6*dw6
		if attn6 > 0 {
			attn6 *= attn6
			value += attn6 * attn6 * s.extrapolate4(xsb+1, ysb+0, zsb+1, wsb+0, dx6, dy6, dz6, dw6)
		}

		// Contribution (1,0,0,1)
		dx7 := dx0 - 1 - 2*squishConstant4D
		dy7 := dy0 - 0 - 2*squishConstant4D
		dz7 := dz0 - 0 - 2*squishConstant4D
		dw7 := dw0 - 1 - 2*squishConstant4D
		attn7 := 2 - dx7*dx7 - dy7*dy7 - dz7*dz7 - dw7*dw7
		if attn7 > 0 {
			attn7 *= attn7
			value += attn7 * attn7 * s.extrapolate4(xsb+1, ysb+0, zsb+0, wsb+1, dx7, dy7, dz7, dw7)
		}

		// Contribution (0,1,1,0)
		dx8 := dx0 - 0 - 2*squishConstant4D
		dy8 := dy0 - 1 - 2*squishConstant4D
		dz8 := dz0 - 1 - 2*squishConstant4D
		dw8 := dw0 - 0 - 2*squishConstant4D
		attn8 := 2 - dx8*dx8 - dy8*dy8 - dz8*dz8 - dw8*dw8
		if attn8 > 0 {
			attn8 *= attn8
			value += attn8 * attn8 * s.extrapolate4(xsb+0, ysb+1, zsb+1, wsb+0, dx8, dy8, dz8, dw8)
		}

		// Contribution (0,1,0,1)
		dx9 := dx0 - 0 - 2*squishConstant4D
		dy9 := dy0 - 1 - 2*squishConstant4D
		dz9 := dz0 - 0 - 2*squishConstant4D
		dw9 := dw0 - 1 - 2*squishConstant4D
		attn9 := 2 - dx9*dx9 - dy9*dy9 - dz9*dz9 - dw9*dw9
		if attn9 > 0 {
			attn9 *= attn9
			value += attn9 * attn9 * s.extrapolate4(xsb+0, ysb+1, zsb+0, wsb+1, dx9, dy9, dz9, dw9)
		}

		// Contribution (0,0,1,1)
		dx10 := dx0 - 0 - 2*squishConstant4D
		dy10 := dy0 - 0 - 2*squishConstant4D
		dz10 := dz0 - 1 - 2*squishConstant4D
		dw10 := dw0 - 1 - 2*squishConstant4D
		attn10 := 2 - dx10*dx10 - dy10*dy10 - dz10*dz10 - dw10*dw10
		if attn10 > 0 {
			attn10 *= attn10
			value += attn10 * attn10 * s.extrapolate4(xsb+0, ysb+0, zsb+1, wsb+1, dx10, dy10, dz10, dw10)
		}
	} else { // We're inside the second dispentachoron (Rectified 4-Simplex)
		var aScore, bScore float64
		var aPoint, bPoint byte
		var aIsBiggerSide bool = true
		var bIsBiggerSide bool = true

		// Decide between (0,0,1,1) and (1,1,0,0)
		if xins+yins < zins+wins {
			aScore = xins + yins
			aPoint = 0x0C
		} else {
			aScore = zins + wins
			aPoint = 0x03
		}

		// Decide between (0,1,0,1) and (1,0,1,0)
		if xins+zins < yins+wins {
			bScore = xins + zins
			bPoint = 0x0A
		} else {
			bScore = yins + wins
			bPoint = 0x05
		}

		// Closer between (0,1,1,0) and (1,0,0,1) will replace the further of a and b, if closer.
		if xins+wins < yins+zins {
			score := xins + wins
			if aScore <= bScore && score < bScore {
				bScore = score
				bPoint = 0x06
			} else if aScore > bScore && score < aScore {
				aScore = score
				aPoint = 0x06
			}
		} else {
			score := yins + zins
			if aScore <= bScore && score < bScore {
				bScore = score
				bPoint = 0x09
			} else if aScore > bScore && score < aScore {
				aScore = score
				aPoint = 0x09
			}
		}

		// Decide if (0,1,1,1) is closer.
		p1 := 3 - inSum + xins
		if aScore <= bScore && p1 < bScore {
			bScore = p1
			bPoint = 0x0E
			bIsBiggerSide = false
		} else if aScore > bScore && p1 < aScore {
			aScore = p1
			aPoint = 0x0E
			aIsBiggerSide = false
		}

		// Decide if (1,0,1,1) is closer.
		p2 := 3 - inSum + yins
		if aScore <= bScore && p2 < bScore {
			bScore = p2
			bPoint = 0x0D
			bIsBiggerSide = false
		} else if aScore > bScore && p2 < aScore {
			aScore = p2
			aPoint = 0x0D
			aIsBiggerSide = false
		}

		// Decide if (1,1,0,1) is closer.
		p3 := 3 - inSum + zins
		if aScore <= bScore && p3 < bScore {
			bScore = p3
			bPoint = 0x0B
			bIsBiggerSide = false
		} else if aScore > bScore && p3 < aScore {
			aScore = p3
			aPoint = 0x0B
			aIsBiggerSide = false
		}

		// Decide if (1,1,1,0) is closer.
		p4 := 3 - inSum + wins
		if aScore <= bScore && p4 < bScore {
			bScore = p4
			bPoint = 0x07
			bIsBiggerSide = false
		} else if aScore > bScore && p4 < aScore {
			aScore = p4
			aPoint = 0x07
			aIsBiggerSide = false
		}

		// Where each of the two closest points are determines how the extra three vertices are calculated.
		if aIsBiggerSide == bIsBiggerSide {
			if aIsBiggerSide { // Both closest points on the bigger side
				c1 := aPoint & bPoint
				c2 := aPoint | bPoint

				// Two contributions are permutations of (0,0,0,1) and (0,0,0,2) based on c1
				xsv_ext1 = xsb
				xsv_ext0 = xsv_ext1
				ysv_ext1 = ysb
				ysv_ext0 = ysv_ext1
				zsv_ext1 = zsb
				zsv_ext0 = zsv_ext1
				wsv_ext1 = wsb
				wsv_ext0 = wsv_ext1
				dx_ext0 = dx0 - squishConstant4D
				dy_ext0 = dy0 - squishConstant4D
				dz_ext0 = dz0 - squishConstant4D
				dw_ext0 = dw0 - squishConstant4D
				dx_ext1 = dx0 - 2*squishConstant4D
				dy_ext1 = dy0 - 2*squishConstant4D
				dz_ext1 = dz0 - 2*squishConstant4D
				dw_ext1 = dw0 - 2*squishConstant4D
				if (c1 & 0x01) != 0 {
					xsv_ext0 += 1
					dx_ext0 -= 1
					xsv_ext1 += 2
					dx_ext1 -= 2
				} else if (c1 & 0x02) != 0 {
					ysv_ext0 += 1
					dy_ext0 -= 1
					ysv_ext1 += 2
					dy_ext1 -= 2
				} else if (c1 & 0x04) != 0 {
					zsv_ext0 += 1
					dz_ext0 -= 1
					zsv_ext1 += 2
					dz_ext1 -= 2
				} else {
					wsv_ext0 += 1
					dw_ext0 -= 1
					wsv_ext1 += 2
					dw_ext1 -= 2
				}

				// One contribution is a permutation of (1,1,1,-1) based on c2
				xsv_ext2 = xsb + 1
				ysv_ext2 = ysb + 1
				zsv_ext2 = zsb + 1
				wsv_ext2 = wsb + 1
				dx_ext2 = dx0 - 1 - 2*squishConstant4D
				dy_ext2 = dy0 - 1 - 2*squishConstant4D
				dz_ext2 = dz0 - 1 - 2*squishConstant4D
				dw_ext2 = dw0 - 1 - 2*squishConstant4D
				if (c2 & 0x01) == 0 {
					xsv_ext2 -= 2
					dx_ext2 += 2
				} else if (c2 & 0x02) == 0 {
					ysv_ext2 -= 2
					dy_ext2 += 2
				} else if (c2 & 0x04) == 0 {
					zsv_ext2 -= 2
					dz_ext2 += 2
				} else {
					wsv_ext2 -= 2
					dw_ext2 += 2
				}
			} else { // Both closest points on the smaller side
				// One of the two extra points is (1,1,1,1)
				xsv_ext2 = xsb + 1
				ysv_ext2 = ysb + 1
				zsv_ext2 = zsb + 1
				wsv_ext2 = wsb + 1
				dx_ext2 = dx0 - 1 - 4*squishConstant4D
				dy_ext2 = dy0 - 1 - 4*squishConstant4D
				dz_ext2 = dz0 - 1 - 4*squishConstant4D
				dw_ext2 = dw0 - 1 - 4*squishConstant4D

				// Other two points are based on the shared axes.
				c := aPoint & bPoint

				if (c & 0x01) != 0 {
					xsv_ext0 = xsb + 2
					xsv_ext1 = xsb + 1
					dx_ext0 = dx0 - 2 - 3*squishConstant4D
					dx_ext1 = dx0 - 1 - 3*squishConstant4D
				} else {
					xsv_ext1 = xsb
					xsv_ext0 = xsv_ext1
					dx_ext1 = dx0 - 3*squishConstant4D
					dx_ext0 = dx_ext1
				}

				if (c & 0x02) != 0 {
					ysv_ext1 = ysb + 1
					ysv_ext0 = ysv_ext1
					dy_ext1 = dy0 - 1 - 3*squishConstant4D
					dy_ext0 = dy_ext1
					if (c & 0x01) == 0 {
						ysv_ext0 += 1
						dy_ext0 -= 1
					} else {
						ysv_ext1 += 1
						dy_ext1 -= 1
					}
				} else {
					ysv_ext1 = ysb
					ysv_ext0 = ysv_ext1
					dy_ext1 = dy0 - 3*squishConstant4D
					dy_ext0 = dy_ext1
				}

				if (c & 0x04) != 0 {
					zsv_ext1 = zsb + 1
					zsv_ext0 = zsv_ext1
					dz_ext1 = dz0 - 1 - 3*squishConstant4D
					dz_ext0 = dz_ext1
					if (c & 0x03) == 0 {
						zsv_ext0 += 1
						dz_ext0 -= 1
					} else {
						zsv_ext1 += 1
						dz_ext1 -= 1
					}
				} else {
					zsv_ext1 = zsb
					zsv_ext0 = zsv_ext1
					dz_ext1 = dz0 - 3*squishConstant4D
					dz_ext0 = dz_ext1
				}

				if (c & 0x08) != 0 {
					wsv_ext0 = wsb + 1
					wsv_ext1 = wsb + 2
					dw_ext0 = dw0 - 1 - 3*squishConstant4D
					dw_ext1 = dw0 - 2 - 3*squishConstant4D
				} else {
					wsv_ext1 = wsb
					wsv_ext0 = wsv_ext1
					dw_ext1 = dw0 - 3*squishConstant4D
					dw_ext0 = dw_ext1
				}
			}
		} else { // One point on each "side"
			var c1, c2 byte
			if aIsBiggerSide {
				c1 = aPoint
				c2 = bPoint
			} else {
				c1 = bPoint
				c2 = aPoint
			}

			// Two contributions are the bigger-sided point with each 1 replaced with 2.
			if (c1 & 0x01) != 0 {
				xsv_ext0 = xsb + 2
				xsv_ext1 = xsb + 1
				dx_ext0 = dx0 - 2 - 3*squishConstant4D
				dx_ext1 = dx0 - 1 - 3*squishConstant4D
			} else {
				xsv_ext1 = xsb
				xsv_ext0 = xsv_ext1
				dx_ext1 = dx0 - 3*squishConstant4D
				dx_ext0 = dx_ext1
			}

			if (c1 & 0x02) != 0 {
				ysv_ext1 = ysb + 1
				ysv_ext0 = ysv_ext1
				dy_ext1 = dy0 - 1 - 3*squishConstant4D
				dy_ext0 = dy_ext1
				if (c1 & 0x01) == 0 {
					ysv_ext0 += 1
					dy_ext0 -= 1
				} else {
					ysv_ext1 += 1
					dy_ext1 -= 1
				}
			} else {
				ysv_ext1 = ysb
				ysv_ext0 = ysv_ext1
				dy_ext1 = dy0 - 3*squishConstant4D
				dy_ext0 = dy_ext1
			}

			if (c1 & 0x04) != 0 {
				zsv_ext1 = zsb + 1
				zsv_ext0 = zsv_ext1
				dz_ext1 = dz0 - 1 - 3*squishConstant4D
				dz_ext0 = dz_ext1
				if (c1 & 0x03) == 0 {
					zsv_ext0 += 1
					dz_ext0 -= 1
				} else {
					zsv_ext1 += 1
					dz_ext1 -= 1
				}
			} else {
				zsv_ext1 = zsb
				zsv_ext0 = zsv_ext1
				dz_ext1 = dz0 - 3*squishConstant4D
				dz_ext0 = dz_ext1
			}

			if (c1 & 0x08) != 0 {
				wsv_ext0 = wsb + 1
				wsv_ext1 = wsb + 2
				dw_ext0 = dw0 - 1 - 3*squishConstant4D
				dw_ext1 = dw0 - 2 - 3*squishConstant4D
			} else {
				wsv_ext1 = wsb
				wsv_ext0 = wsv_ext1
				dw_ext1 = dw0 - 3*squishConstant4D
				dw_ext0 = dw_ext1
			}

			// One contribution is a permutation of (1,1,1,-1) based on the smaller-sided point
			xsv_ext2 = xsb + 1
			ysv_ext2 = ysb + 1
			zsv_ext2 = zsb + 1
			wsv_ext2 = wsb + 1
			dx_ext2 = dx0 - 1 - 2*squishConstant4D
			dy_ext2 = dy0 - 1 - 2*squishConstant4D
			dz_ext2 = dz0 - 1 - 2*squishConstant4D
			dw_ext2 = dw0 - 1 - 2*squishConstant4D
			if (c2 & 0x01) == 0 {
				xsv_ext2 -= 2
				dx_ext2 += 2
			} else if (c2 & 0x02) == 0 {
				ysv_ext2 -= 2
				dy_ext2 += 2
			} else if (c2 & 0x04) == 0 {
				zsv_ext2 -= 2
				dz_ext2 += 2
			} else {
				wsv_ext2 -= 2
				dw_ext2 += 2
			}
		}

		// Contribution (1,1,1,0)
		dx4 := dx0 - 1 - 3*squishConstant4D
		dy4 := dy0 - 1 - 3*squishConstant4D
		dz4 := dz0 - 1 - 3*squishConstant4D
		dw4 := dw0 - 3*squishConstant4D
		attn4 := 2 - dx4*dx4 - dy4*dy4 - dz4*dz4 - dw4*dw4
		if attn4 > 0 {
			attn4 *= attn4
			value += attn4 * attn4 * s.extrapolate4(xsb+1, ysb+1, zsb+1, wsb+0, dx4, dy4, dz4, dw4)
		}

		// Contribution (1,1,0,1)
		dx3 := dx4
		dy3 := dy4
		dz3 := dz0 - 3*squishConstant4D
		dw3 := dw0 - 1 - 3*squishConstant4D
		attn3 := 2 - dx3*dx3 - dy3*dy3 - dz3*dz3 - dw3*dw3
		if attn3 > 0 {
			attn3 *= attn3
			value += attn3 * attn3 * s.extrapolate4(xsb+1, ysb+1, zsb+0, wsb+1, dx3, dy3, dz3, dw3)
		}

		// Contribution (1,0,1,1)
		dx2 := dx4
		dy2 := dy0 - 3*squishConstant4D
		dz2 := dz4
		dw2 := dw3
		attn2 := 2 - dx2*dx2 - dy2*dy2 - dz2*dz2 - dw2*dw2
		if attn2 > 0 {
			attn2 *= attn2
			value += attn2 * attn2 * s.extrapolate4(xsb+1, ysb+0, zsb+1, wsb+1, dx2, dy2, dz2, dw2)
		}

		// Contribution (0,1,1,1)
		dx1 := dx0 - 3*squishConstant4D
		dz1 := dz4
		dy1 := dy4
		dw1 := dw3
		attn1 := 2 - dx1*dx1 - dy1*dy1 - dz1*dz1 - dw1*dw1
		if attn1 > 0 {
			attn1 *= attn1
			value += attn1 * attn1 * s.extrapolate4(xsb+0, ysb+1, zsb+1, wsb+1, dx1, dy1, dz1, dw1)
		}

		// Contribution (1,1,0,0)
		dx5 := dx0 - 1 - 2*squishConstant4D
		dy5 := dy0 - 1 - 2*squishConstant4D
		dz5 := dz0 - 0 - 2*squishConstant4D
		dw5 := dw0 - 0 - 2*squishConstant4D
		attn5 := 2 - dx5*dx5 - dy5*dy5 - dz5*dz5 - dw5*dw5
		if attn5 > 0 {
			attn5 *= attn5
			value += attn5 * attn5 * s.extrapolate4(xsb+1, ysb+1, zsb+0, wsb+0, dx5, dy5, dz5, dw5)
		}

		// Contribution (1,0,1,0)
		dx6 := dx0 - 1 - 2*squishConstant4D
		dy6 := dy0 - 0 - 2*squishConstant4D
		dz6 := dz0 - 1 - 2*squishConstant4D
		dw6 := dw0 - 0 - 2*squishConstant4D
		attn6 := 2 - dx6*dx6 - dy6*dy6 - dz6*dz6 - dw6*dw6
		if attn6 > 0 {
			attn6 *= attn6
			value += attn6 * attn6 * s.extrapolate4(xsb+1, ysb+0, zsb+1, wsb+0, dx6, dy6, dz6, dw6)
		}

		// Contribution (1,0,0,1)
		dx7 := dx0 - 1 - 2*squishConstant4D
		dy7 := dy0 - 0 - 2*squishConstant4D
		dz7 := dz0 - 0 - 2*squishConstant4D
		dw7 := dw0 - 1 - 2*squishConstant4D
		attn7 := 2 - dx7*dx7 - dy7*dy7 - dz7*dz7 - dw7*dw7
		if attn7 > 0 {
			attn7 *= attn7
			value += attn7 * attn7 * s.extrapolate4(xsb+1, ysb+0, zsb+0, wsb+1, dx7, dy7, dz7, dw7)
		}

		// Contribution (0,1,1,0)
		dx8 := dx0 - 0 - 2*squishConstant4D
		dy8 := dy0 - 1 - 2*squishConstant4D
		dz8 := dz0 - 1 - 2*squishConstant4D
		dw8 := dw0 - 0 - 2*squishConstant4D
		attn8 := 2 - dx8*dx8 - dy8*dy8 - dz8*dz8 - dw8*dw8
		if attn8 > 0 {
			attn8 *= attn8
			value += attn8 * attn8 * s.extrapolate4(xsb+0, ysb+1, zsb+1, wsb+0, dx8, dy8, dz8, dw8)
		}

		// Contribution (0,1,0,1)
		dx9 := dx0 - 0 - 2*squishConstant4D
		dy9 := dy0 - 1 - 2*squishConstant4D
		dz9 := dz0 - 0 - 2*squishConstant4D
		dw9 := dw0 - 1 - 2*squishConstant4D
		attn9 := 2 - dx9*dx9 - dy9*dy9 - dz9*dz9 - dw9*dw9
		if attn9 > 0 {
			attn9 *= attn9
			value += attn9 * attn9 * s.extrapolate4(xsb+0, ysb+1, zsb+0, wsb+1, dx9, dy9, dz9, dw9)
		}

		// Contribution (0,0,1,1)
		dx10 := dx0 - 0 - 2*squishConstant4D
		dy10 := dy0 - 0 - 2*squishConstant4D
		dz10 := dz0 - 1 - 2*squishConstant4D
		dw10 := dw0 - 1 - 2*squishConstant4D
		attn10 := 2 - dx10*dx10 - dy10*dy10 - dz10*dz10 - dw10*dw10
		if attn10 > 0 {
			attn10 *= attn10
			value += attn10 * attn10 * s.extrapolate4(xsb+0, ysb+0, zsb+1, wsb+1, dx10, dy10, dz10, dw10)
		}
	}

	// First extra vertex
	attn_ext0 := 2 - dx_ext0*dx_ext0 - dy_ext0*dy_ext0 - dz_ext0*dz_ext0 - dw_ext0*dw_ext0
	if attn_ext0 > 0 {
		attn_ext0 *= attn_ext0
		value += attn_ext0 * attn_ext0 * s.extrapolate4(xsv_ext0, ysv_ext0, zsv_ext0, wsv_ext0, dx_ext0, dy_ext0, dz_ext0, dw_ext0)
	}

	// Second extra vertex
	attn_ext1 := 2 - dx_ext1*dx_ext1 - dy_ext1*dy_ext1 - dz_ext1*dz_ext1 - dw_ext1*dw_ext1
	if attn_ext1 > 0 {
		attn_ext1 *= attn_ext1
		value += attn_ext1 * attn_ext1 * s.extrapolate4(xsv_ext1, ysv_ext1, zsv_ext1, wsv_ext1, dx_ext1, dy_ext1, dz_ext1, dw_ext1)
	}

	// Third extra vertex
	attn_ext2 := 2 - dx_ext2*dx_ext2 - dy_ext2*dy_ext2 - dz_ext2*dz_ext2 - dw_ext2*dw_ext2
	if attn_ext2 > 0 {
		attn_ext2 *= attn_ext2
		value += attn_ext2 * attn_ext2 * s.extrapolate4(xsv_ext2, ysv_ext2, zsv_ext2, wsv_ext2, dx_ext2, dy_ext2, dz_ext2, dw_ext2)
	}

	return value / normConstant4D
}

func (s *Noise) extrapolate2(xsb, ysb int32, dx, dy float64) float64 {
	index := s.perm[(int32(s.perm[xsb&0xFF])+ysb)&0xFF] & 0x0E
	return float64(gradients2D[index])*dx + float64(gradients2D[index+1])*dy
}

func (s *Noise) extrapolate3(xsb, ysb, zsb int32, dx, dy, dz float64) float64 {
	index := s.permGradIndex3D[(int32(s.perm[(int32(s.perm[xsb&0xFF])+ysb)&0xFF])+zsb)&0xFF]
	return float64(gradients3D[index])*dx + float64(gradients3D[index+1])*dy + float64(gradients3D[index+2])*dz
}

func (s *Noise) extrapolate4(xsb, ysb, zsb, wsb int32, dx, dy, dz, dw float64) float64 {
	index := s.perm[(int32(s.perm[(int32(s.perm[(int32(s.perm[xsb&0xFF])+ysb)&0xFF])+zsb)&0xFF])+wsb)&0xFF] & 0xFC
	return float64(gradients4D[index])*dx + float64(gradients4D[index+1])*dy + float64(gradients4D[index+2])*dz + float64(gradients4D[index+3])*dw
}

// Gradients for 2D. They approximate the directions to the
// vertices of an octagon from the center.
var gradients2D = []int8{
	5, 2, 2, 5,
	-5, 2, -2, 5,
	5, -2, 2, -5,
	-5, -2, -2, -5,
}

// Gradients for 3D. They approximate the directions to the
// vertices of a rhombicuboctahedron from the center, skewed so
// that the triangular and square facets can be inscribed inside
// circles of the same radius.
var gradients3D = []int8{
	-11, 4, 4, -4, 11, 4, -4, 4, 11,
	11, 4, 4, 4, 11, 4, 4, 4, 11,
	-11, -4, 4, -4, -11, 4, -4, -4, 11,
	11, -4, 4, 4, -11, 4, 4, -4, 11,
	-11, 4, -4, -4, 11, -4, -4, 4, -11,
	11, 4, -4, 4, 11, -4, 4, 4, -11,
	-11, -4, -4, -4, -11, -4, -4, -4, -11,
	11, -4, -4, 4, -11, -4, 4, -4, -11,
}

// Gradients for 4D. They approximate the directions to the
// vertices of a disprismatotesseractihexadecachoron from the center,
// skewed so that the tetrahedral and cubic facets can be inscribed inside
// spheres of the same radius.
var gradients4D = []int8{
	3, 1, 1, 1, 1, 3, 1, 1, 1, 1, 3, 1, 1, 1, 1, 3,
	-3, 1, 1, 1, -1, 3, 1, 1, -1, 1, 3, 1, -1, 1, 1, 3,
	3, -1, 1, 1, 1, -3, 1, 1, 1, -1, 3, 1, 1, -1, 1, 3,
	-3, -1, 1, 1, -1, -3, 1, 1, -1, -1, 3, 1, -1, -1, 1, 3,
	3, 1, -1, 1, 1, 3, -1, 1, 1, 1, -3, 1, 1, 1, -1, 3,
	-3, 1, -1, 1, -1, 3, -1, 1, -1, 1, -3, 1, -1, 1, -1, 3,
	3, -1, -1, 1, 1, -3, -1, 1, 1, -1, -3, 1, 1, -1, -1, 3,
	-3, -1, -1, 1, -1, -3, -1, 1, -1, -1, -3, 1, -1, -1, -1, 3,
	3, 1, 1, -1, 1, 3, 1, -1, 1, 1, 3, -1, 1, 1, 1, -3,
	-3, 1, 1, -1, -1, 3, 1, -1, -1, 1, 3, -1, -1, 1, 1, -3,
	3, -1, 1, -1, 1, -3, 1, -1, 1, -1, 3, -1, 1, -1, 1, -3,
	-3, -1, 1, -1, -1, -3, 1, -1, -1, -1, 3, -1, -1, -1, 1, -3,
	3, 1, -1, -1, 1, 3, -1, -1, 1, 1, -3, -1, 1, 1, -1, -3,
	-3, 1, -1, -1, -1, 3, -1, -1, -1, 1, -3, -1, -1, 1, -1, -3,
	3, -1, -1, -1, 1, -3, -1, -1, 1, -1, -3, -1, 1, -1, -1, -3,
	-3, -1, -1, -1, -1, -3, -1, -1, -1, -1, -3, -1, -1, -1, -1, -3,
}
