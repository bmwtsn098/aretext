package segment

import (
	"unicode"
)

// Segment represents a sequence of runes from a larger text (for example a grapheme cluster).
type Segment struct {
	runes []rune
}

// NewSegment returns a new, empty segment.
func NewSegment() *Segment {
	return &Segment{runes: make([]rune, 0, 1)}
}

// Clear removes all runes from the segment.
func (seg *Segment) Clear() *Segment {
	seg.runes = seg.runes[:0]
	return seg
}

// Extend appends all runes from the slice.
func (seg *Segment) Extend(runes []rune) *Segment {
	for _, r := range runes {
		seg.runes = append(seg.runes, r)
	}
	return seg
}

// ReverseRunes reverses the order of the runes in the segment.
func (seg *Segment) ReverseRunes() *Segment {
	i := 0
	j := len(seg.runes) - 1
	for i < j {
		seg.runes[i], seg.runes[j] = seg.runes[j], seg.runes[i]
		i++
		j--
	}
	return seg
}

// Clone returns a copy of the segment.
func (seg *Segment) Clone() *Segment {
	runes := make([]rune, len(seg.runes))
	copy(runes, seg.runes)
	return &Segment{runes}
}

// NumRunes returns the number of runes in the segment.
func (seg *Segment) NumRunes() uint64 {
	return uint64(len(seg.runes))
}

// Runes returns the runes contained in the segment.
// Callers should not modify the returned slice.
func (seg *Segment) Runes() []rune {
	return seg.runes
}

// HasNewline checks whether a segment contains a line feed rune.
func (seg *Segment) HasNewline() bool {
	for i := len(seg.runes) - 1; i >= 0; i-- {
		if seg.runes[i] == '\n' {
			return true
		}
	}
	return false
}

// SegmentIsWhitespace checks whether a segment contains all whitespace runes (spaces, tabs, etc).
func (seg *Segment) IsWhitespace() bool {
	for _, r := range seg.runes {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}