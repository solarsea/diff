// Package diff provides a diff algorithm implementation
// for finite, indexable sequences with comparable elements.
package diff // import "github.com/solarsea/diff"

import (
	bits "github.com/solarsea/bits"
)

// Interface abstracts the required knowledge to perform a diff
// on any two fixed-length sequences with comparable elements.
type Interface interface {
	// The sequences' lengths
	Len() (int, int)
	// True when the sequences' elements at those indices are equal
	Equal(int, int) bool
}

// A Mark struct marks a length in sequence starting with an offset
type Mark struct {
	From   int
	Length int
}

// A Delta struct is the result of a Diff operation
type Delta struct {
	Added   []Mark
	Removed []Mark
}

// Diffs the provided data and returns e Delta struct
// with added entries' indices in the second sequence and removed from the first
func Diff(data Interface) Delta {
	var len1, len2 = data.Len()
	var mx *matrix = &matrix{v: bits.NewBit(uint(len1 * len2)), lenX: len1, lenY: len2}
	mx.matches = make(map[point]int)

	for i := 0; i < len1; i++ {
		for j := 0; j < len2; j++ {
			mx.v.Poke(mx.at(point{i, j}), data.Equal(i, j))
		}
	}

	return mx.recursiveDiff(box{point{0, 0}, len1, len2})
}

type point struct {
	x, y   int
}

type match struct {
	point
	length int
}

type box struct {
	point
	lenX, lenY int
}

// A helper structure that stores absolute dimension along a linear bit vector
// so that it can always properly translate (x, y) -> z on the vector
type matrix struct {
	v          bits.Vector
	lenX, lenY int
	matches map[point]int
}

// Translates (x, y) to an absolute position on the bit vector
func (mx *matrix) at(p point) uint {
	return uint(p.y + (p.x * mx.lenY))
}

func (mx *matrix) recursiveDiff(bounds box) Delta {
	var m match = mx.largest(bounds)

	if m.length == 0 { // Recursion terminates
		var immediate Delta
		if bounds.lenY-bounds.y > 0 {
			immediate.Added = []Mark{Mark{bounds.y, bounds.lenY}}
		}
		if bounds.lenX-bounds.x > 0 {
			immediate.Removed = []Mark{Mark{bounds.x, bounds.lenX}}
		}
		return immediate
	}

	var left Delta = mx.recursiveDiff(box{point{bounds.x, bounds.y}, m.x, m.y})
	var right Delta = mx.recursiveDiff(box{point{m.x + m.length, m.y + m.length}, bounds.lenX, bounds.lenY})

	var result Delta

	result.Added = append(left.Added, right.Added...)
	result.Removed = append(left.Removed, right.Removed...)

	return result
}

// Finds the largest common substring by looking at the provided match matrix
// starting from (bounds.x, bounds.y) with lengths bounds.lenX, bounds.lenY
func (mx *matrix) largest(bounds box) match {
	var result match

	// Look for LCS in the too-right half, including the main diagonal
	for i := bounds.x; i < bounds.lenX && result.length < (bounds.lenX-i); i++ {
		var m match = mx.search(box{point{i, bounds.y}, bounds.lenX, bounds.lenY})
		if m.length > result.length {
			result = m
		}
	}

	// Look for LCS in the bottom-left half, excluding the main diagonal
	for j := bounds.y + 1; j < bounds.lenY && result.length < (bounds.lenY-j); j++ {
		var m match = mx.search(box{point{bounds.x, j}, bounds.lenX, bounds.lenY})
		if m.length > result.length {
			result = m
		}
	}
	return result
}

// Searches the main diagonal for the longest sequential match line
func (mx *matrix) search(bounds box) (result match) {
	var inMatch bool
	var m match
	for step := 0; step+bounds.x < bounds.lenX && step+bounds.y < bounds.lenY; {
		var current point = point{step+bounds.x, step+bounds.y}
		if length, found := mx.matches[current]; found {
			if length > result.length {
				result.point = current
				result.length = length
			}
			step += length
			continue
		}
		if mx.v.Peek(mx.at(current)) {
			if !inMatch { // Create a new current record if there is none ...
				inMatch, m.point, m.length = true, current, 1
			} else { // ... otherwise just increment the existing
				m.length++
			}
			// Update the length in the cache
			mx.matches[m.point] = m.length
			if m.length > result.length {
				result = m // Store it if it is longer ...
			}
		} else { // End of current of match
			inMatch = false // ... and reset the current one
		}
		step++
	}
	return
}

// A diff.Interface implementation with plugable Equal function
type impl struct {
	len1, len2 int
	equal      func(i, j int) bool
}

// Required per diff.Interface
func (d impl) Len() (int, int) {
	return d.len1, d.len2
}

// Required per diff.Interface
func (d impl) Equal(i, j int) bool {
	return d.equal(i, j)
}

// Returns a diff.Interface implementation
// for the specified lengths and equal function
func WithEqual(len1 int, len2 int, equal func(int, int) bool) Interface {
	return impl{len1: len1, len2: len2, equal: equal}
}
