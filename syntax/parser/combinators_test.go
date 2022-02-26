package parser

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aretext/aretext/text"
)

func TestMaybeBefore(t *testing.T) {
	// Parse consecutive numerals as numbers.
	firstParseFunc := func(iter TrackingRuneIter, state State) Result {
		var n uint64
		for {
			r, err := iter.NextRune()
			if err != nil || r < '0' || r > '9' {
				break
			}
			n++
		}
		return Result{
			NumConsumed: n,
			ComputedTokens: []ComputedToken{
				{
					Length: n,
					Role:   TokenRoleNumber,
				},
			},
			NextState: state,
		}
	}

	// Parse alpha characters as identifiers.
	secondParseFunc := func(iter TrackingRuneIter, state State) Result {
		var n uint64
		for {
			r, err := iter.NextRune()
			if err != nil || r < 'A' || r > 'z' {
				break
			}
			n++
		}
		return Result{
			NumConsumed: n,
			ComputedTokens: []ComputedToken{
				{
					Length: n,
					Role:   TokenRoleIdentifier,
				},
			},
			NextState: state,
		}
	}

	// Alpha characters, optionally prefixed with spaces.
	combinedParseFunc := Func(firstParseFunc).MaybeBefore(Func(secondParseFunc))

	testCases := []struct {
		name     string
		text     string
		expected []Token
	}{
		{
			name: "only second parse func",
			text: "abc",
			expected: []Token{
				{StartPos: 0, EndPos: 3, Role: TokenRoleIdentifier},
			},
		},
		{
			name: "first and second parse func",
			text: "1234abc",
			expected: []Token{
				{StartPos: 0, EndPos: 4, Role: TokenRoleNumber},
				{StartPos: 4, EndPos: 7, Role: TokenRoleIdentifier},
			},
		},
		{
			name:     "only first parse func",
			text:     "1234",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tree, err := text.NewTreeFromString(tc.text)
			require.NoError(t, err)

			p := New(combinedParseFunc)
			p.ParseAll(tree)
			tokens := p.TokensIntersectingRange(0, math.MaxUint64)
			assert.Equal(t, tc.expected, tokens)
		})
	}

}

func TestThenCombinatorShiftTokens(t *testing.T) {
	// Parse up to ":" as an identifier.
	firstParseFunc := func(iter TrackingRuneIter, state State) Result {
		var n uint64
		for {
			r, err := iter.NextRune()
			if err != nil || r == ':' {
				break
			}
			n++
		}
		return Result{
			NumConsumed: n,
			NextState:   state,
			ComputedTokens: []ComputedToken{
				{
					Length: n,
					Role:   TokenRoleIdentifier,
				},
			},
		}
	}

	// Parse rest of the string as a number.
	secondParseFunc := func(iter TrackingRuneIter, state State) Result {
		var n uint64
		for {
			_, err := iter.NextRune()
			if err != nil {
				break
			}
			n++
		}
		return Result{
			NumConsumed: n,
			NextState:   state,
			ComputedTokens: []ComputedToken{
				{
					Length: n,
					Role:   TokenRoleNumber,
				},
			},
		}
	}

	tree, err := text.NewTreeFromString("abc:123")
	require.NoError(t, err)

	combinedParseFunc := Func(firstParseFunc).Then(Func(secondParseFunc))
	p := New(combinedParseFunc)
	p.ParseAll(tree)
	tokens := p.TokensIntersectingRange(0, math.MaxUint64)
	expectedTokens := []Token{
		{StartPos: 0, EndPos: 3, Role: TokenRoleIdentifier},
		{StartPos: 3, EndPos: 7, Role: TokenRoleNumber},
	}
	assert.Equal(t, expectedTokens, tokens)
}