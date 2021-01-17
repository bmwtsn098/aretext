package exec

import (
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/aretext/aretext/config"
	"github.com/aretext/aretext/syntax"
	"github.com/aretext/aretext/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestFile(t *testing.T, contents string) (path string, cleanup func()) {
	f, err := ioutil.TempFile(os.TempDir(), "aretext-")
	require.NoError(t, err)
	defer f.Close()

	_, err = io.WriteString(f, contents)
	require.NoError(t, err)

	cleanup = func() { os.Remove(f.Name()) }
	return f.Name(), cleanup
}

func TestLoadDocumentMutator(t *testing.T) {
	// Start with an empty document.
	state := NewEditorState(100, 100, config.RuleSet{})
	assert.Equal(t, "", state.documentBuffer.textTree.String())
	assert.Equal(t, "", state.FileWatcher().Path())

	// Load a new document.
	path, cleanup := createTestFile(t, "abcd")
	defer cleanup()
	NewLoadDocumentMutator(path, true, false).Mutate(state)
	defer state.FileWatcher().Stop()

	// Expect that the text and watcher are installed.
	assert.Equal(t, "abcd", state.documentBuffer.textTree.String())
	assert.Equal(t, path, state.FileWatcher().Path())
}

func TestLoadDocumentMutatorSameFile(t *testing.T) {
	// Load the initial document.
	path, cleanup := createTestFile(t, "abcd\nefghi\njklmnop\nqrst")
	defer cleanup()
	state := NewEditorState(5, 3, config.RuleSet{})
	NewLoadDocumentMutator(path, true, false).Mutate(state)
	state.documentBuffer.cursor.position = 22

	// Scroll to cursor at end of document.
	NewScrollToCursorMutator().Mutate(state)
	assert.Equal(t, uint64(16), state.documentBuffer.view.textOrigin)

	// Set the syntax.
	NewSetSyntaxMutator(syntax.LanguageJson).Mutate(state)
	assert.Equal(t, syntax.LanguageJson, state.documentBuffer.syntaxLanguage)

	// Update the file with shorter text and reload.
	err := ioutil.WriteFile(path, []byte("ab"), 0644)
	require.NoError(t, err)
	NewReloadDocumentMutator(false).Mutate(state)
	defer state.fileWatcher.Stop()

	// Expect that the cursor moved back to the end of the text,
	// the view scrolled to make the cursor visible,
	// and the syntax language is preserved.
	assert.Equal(t, "ab", state.documentBuffer.textTree.String())
	assert.Equal(t, uint64(1), state.documentBuffer.cursor.position)
	assert.Equal(t, uint64(0), state.documentBuffer.view.textOrigin)
	assert.Equal(t, syntax.LanguageJson, state.documentBuffer.syntaxLanguage)
}

func TestLoadDocumentMutatorDifferentFile(t *testing.T) {
	// Load the initial document.
	path, cleanup := createTestFile(t, "abcd\nefghi\njklmnop\nqrst")
	defer cleanup()
	state := NewEditorState(5, 3, config.RuleSet{})
	NewLoadDocumentMutator(path, true, false).Mutate(state)
	state.documentBuffer.cursor.position = 22

	// Scroll to cursor at end of document.
	NewScrollToCursorMutator().Mutate(state)
	assert.Equal(t, uint64(16), state.documentBuffer.view.textOrigin)

	// Set the syntax.
	NewSetSyntaxMutator(syntax.LanguageJson).Mutate(state)
	assert.Equal(t, syntax.LanguageJson, state.documentBuffer.syntaxLanguage)

	// Load a new document with a shorter text.
	path2, cleanup2 := createTestFile(t, "ab")
	defer cleanup2()
	NewLoadDocumentMutator(path2, true, false).Mutate(state)
	defer state.fileWatcher.Stop()

	// Expect that the cursor, view, and syntax are reset.
	assert.Equal(t, "ab", state.documentBuffer.textTree.String())
	assert.Equal(t, uint64(0), state.documentBuffer.cursor.position)
	assert.Equal(t, uint64(0), state.documentBuffer.view.textOrigin)
	assert.Equal(t, syntax.LanguageUndefined, state.documentBuffer.syntaxLanguage)
}

func TestLoadDocumentMutatorShowStatus(t *testing.T) {
	// Start with an empty document.
	state := NewEditorState(100, 100, config.RuleSet{})

	// Load a document, expect success msg.
	path, cleanup := createTestFile(t, "")
	NewLoadDocumentMutator(path, true, true).Mutate(state)
	defer state.fileWatcher.Stop()
	assert.Contains(t, state.statusMsg.Text, "Opened")
	assert.Equal(t, StatusMsgStyleSuccess, state.statusMsg.Style)

	// Delete the test file.
	cleanup()

	// Load a non-existent path, expect error msg.
	NewLoadDocumentMutator(path, true, true).Mutate(state)
	defer state.fileWatcher.Stop()
	assert.Contains(t, state.statusMsg.Text, "Could not open")
	assert.Equal(t, StatusMsgStyleError, state.statusMsg.Style)
}

func TestSaveDocumentMutator(t *testing.T) {
	// Start with an empty document.
	state := NewEditorState(100, 100, config.RuleSet{})

	// Load an existing document.
	path, cleanup := createTestFile(t, "")
	defer cleanup()
	NewLoadDocumentMutator(path, true, true).Mutate(state)
	defer state.fileWatcher.Stop()

	// Modify and save the document
	NewCompositeMutator([]Mutator{
		NewInsertRuneMutator('x'),
		NewSaveDocumentMutator(true),
	}).Mutate(state)
	defer state.fileWatcher.Stop()

	// Expect a success message.
	assert.Contains(t, state.statusMsg.Text, "Saved")
	assert.Equal(t, StatusMsgStyleSuccess, state.statusMsg.Style)

	// Check that the changes were persisted
	contents, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "x\n", string(contents))
}

func TestSaveDocumentMutatorFileChanged(t *testing.T) {
	testCases := []struct {
		name        string
		force       bool
		expectSaved bool
	}{
		{
			name:        "force should save",
			force:       true,
			expectSaved: true,
		},
		{
			name:        "no force should error",
			force:       false,
			expectSaved: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Load the initial document.
			path, cleanup := createTestFile(t, "")
			defer cleanup()
			state := NewEditorState(100, 100, config.RuleSet{})
			NewLoadDocumentMutator(path, true, true).Mutate(state)

			// Modify the file.
			f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			require.NoError(t, err)
			defer f.Close()
			_, err = io.WriteString(f, "test")
			require.NoError(t, err)

			// Wait for the watcher to detect the change.
			select {
			case <-state.fileWatcher.ChangedChan():
				break
			case <-time.After(time.Second * 10):
				assert.Fail(t, "Timed out waiting for change")
				return
			}

			// Attempt to save the document.
			NewSaveDocumentMutator(tc.force).Mutate(state)

			// Retrieve the file contents
			contents, err := ioutil.ReadFile(path)
			require.NoError(t, err)

			if tc.expectSaved {
				assert.Equal(t, StatusMsgStyleSuccess, state.statusMsg.Style)
				assert.Contains(t, state.statusMsg.Text, "Saved")
				assert.Equal(t, "\n", string(contents))
			} else {
				assert.Equal(t, StatusMsgStyleError, state.statusMsg.Style)
				assert.Contains(t, state.statusMsg.Text, "changed since last save")
				assert.Equal(t, "test", string(contents))
			}
		})
	}
}

func TestCursorMutator(t *testing.T) {
	textTree, err := text.NewTreeFromString("abcd")
	require.NoError(t, err)
	state := NewEditorState(100, 100, config.RuleSet{})
	state.documentBuffer.textTree = textTree
	state.documentBuffer.cursor.position = 2
	mutator := NewCursorMutator(NewCharInLineLocator(text.ReadDirectionForward, 1, false))
	mutator.Mutate(state)
	assert.Equal(t, uint64(3), state.documentBuffer.cursor.position)
}

func TestInsertRuneMutator(t *testing.T) {
	testCases := []struct {
		name           string
		inputString    string
		initialCursor  cursorState
		insertRune     rune
		expectedCursor cursorState
		expectedText   string
	}{
		{
			name:           "insert into empty string",
			inputString:    "",
			initialCursor:  cursorState{position: 0},
			insertRune:     'x',
			expectedCursor: cursorState{position: 1},
			expectedText:   "x",
		},
		{
			name:           "insert in middle of string",
			inputString:    "abcd",
			initialCursor:  cursorState{position: 1},
			insertRune:     'x',
			expectedCursor: cursorState{position: 2},
			expectedText:   "axbcd",
		},
		{
			name:           "insert at end of string",
			inputString:    "abcd",
			initialCursor:  cursorState{position: 4},
			insertRune:     'x',
			expectedCursor: cursorState{position: 5},
			expectedText:   "abcdx",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			textTree, err := text.NewTreeFromString(tc.inputString)
			require.NoError(t, err)
			state := NewEditorState(100, 100, config.RuleSet{})
			state.documentBuffer.textTree = textTree
			state.documentBuffer.cursor = tc.initialCursor
			mutator := NewInsertRuneMutator(tc.insertRune)
			mutator.Mutate(state)
			assert.Equal(t, tc.expectedCursor, state.documentBuffer.cursor)
			assert.Equal(t, tc.expectedText, textTree.String())
		})
	}
}

func TestDeleteMutator(t *testing.T) {
	testCases := []struct {
		name           string
		inputString    string
		initialCursor  cursorState
		locator        CursorLocator
		expectedCursor cursorState
		expectedText   string
	}{
		{
			name:           "delete from empty string",
			inputString:    "",
			initialCursor:  cursorState{position: 0},
			locator:        NewCharInLineLocator(text.ReadDirectionForward, 1, true),
			expectedCursor: cursorState{position: 0},
			expectedText:   "",
		},
		{
			name:           "delete next character at start of string",
			inputString:    "abcd",
			initialCursor:  cursorState{position: 0},
			locator:        NewCharInLineLocator(text.ReadDirectionForward, 1, true),
			expectedCursor: cursorState{position: 0},
			expectedText:   "bcd",
		},
		{
			name:           "delete from end of text",
			inputString:    "abcd",
			initialCursor:  cursorState{position: 3},
			locator:        NewCharInLineLocator(text.ReadDirectionForward, 1, true),
			expectedCursor: cursorState{position: 3},
			expectedText:   "abc",
		},
		{
			name:           "delete multiple characters",
			inputString:    "abcd",
			initialCursor:  cursorState{position: 1},
			locator:        NewCharInLineLocator(text.ReadDirectionForward, 10, true),
			expectedCursor: cursorState{position: 1},
			expectedText:   "a",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			textTree, err := text.NewTreeFromString(tc.inputString)
			require.NoError(t, err)
			state := NewEditorState(100, 100, config.RuleSet{})
			state.documentBuffer.textTree = textTree
			state.documentBuffer.cursor = tc.initialCursor
			mutator := NewDeleteMutator(tc.locator)
			mutator.Mutate(state)
			assert.Equal(t, tc.expectedCursor, state.documentBuffer.cursor)
			assert.Equal(t, tc.expectedText, textTree.String())
		})
	}
}

func TestDeleteLinesMutator(t *testing.T) {
	testCases := []struct {
		name                       string
		inputString                string
		initialCursor              cursorState
		targetLineLocator          CursorLocator
		abortIfTargetIsCurrentLine bool
		expectedCursor             cursorState
		expectedText               string
		expectedUnsavedChanges     bool
	}{
		{
			name:                   "empty",
			inputString:            "",
			initialCursor:          cursorState{position: 0},
			targetLineLocator:      NewRelativeLineStartLocator(text.ReadDirectionForward, 1),
			expectedCursor:         cursorState{position: 0},
			expectedText:           "",
			expectedUnsavedChanges: false,
		},
		{
			name:                   "delete single line",
			inputString:            "abcd",
			initialCursor:          cursorState{position: 2},
			targetLineLocator:      NewCurrentCursorLocator(),
			expectedCursor:         cursorState{position: 0},
			expectedText:           "",
			expectedUnsavedChanges: true,
		},
		{
			name:                       "delete single line, abort if same line",
			inputString:                "abcd",
			initialCursor:              cursorState{position: 2},
			targetLineLocator:          NewCurrentCursorLocator(),
			abortIfTargetIsCurrentLine: true,
			expectedCursor:             cursorState{position: 2},
			expectedText:               "abcd",
			expectedUnsavedChanges:     false,
		},
		{
			name:                   "delete single line, first line",
			inputString:            "abcd\nefgh\nijk",
			initialCursor:          cursorState{position: 2},
			targetLineLocator:      NewCurrentCursorLocator(),
			expectedCursor:         cursorState{position: 0},
			expectedText:           "efgh\nijk",
			expectedUnsavedChanges: true,
		},
		{
			name:                   "delete single line, interior line",
			inputString:            "abcd\nefgh\nijk",
			initialCursor:          cursorState{position: 6},
			targetLineLocator:      NewCurrentCursorLocator(),
			expectedCursor:         cursorState{position: 5},
			expectedText:           "abcd\nijk",
			expectedUnsavedChanges: true,
		},
		{
			name:                   "delete single line, last line",
			inputString:            "abcd\nefgh\nijk",
			initialCursor:          cursorState{position: 12},
			targetLineLocator:      NewCurrentCursorLocator(),
			expectedCursor:         cursorState{position: 5},
			expectedText:           "abcd\nefgh",
			expectedUnsavedChanges: true,
		},
		{
			name:                   "delete empty line",
			inputString:            "abcd\n\nefgh",
			initialCursor:          cursorState{position: 5},
			targetLineLocator:      NewCurrentCursorLocator(),
			expectedCursor:         cursorState{position: 5},
			expectedText:           "abcd\nefgh",
			expectedUnsavedChanges: true,
		},
		{
			name:                   "delete multiple lines down",
			inputString:            "abcd\nefgh\nijk\nlmnop",
			initialCursor:          cursorState{position: 0},
			targetLineLocator:      NewRelativeLineStartLocator(text.ReadDirectionForward, 2),
			expectedCursor:         cursorState{position: 0},
			expectedText:           "lmnop",
			expectedUnsavedChanges: true,
		},
		{
			name:                   "delete multiple lines down",
			inputString:            "abcd\nefgh\nijk\nlmnop",
			initialCursor:          cursorState{position: 16},
			targetLineLocator:      NewRelativeLineStartLocator(text.ReadDirectionBackward, 2),
			expectedCursor:         cursorState{position: 0},
			expectedText:           "abcd",
			expectedUnsavedChanges: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			textTree, err := text.NewTreeFromString(tc.inputString)
			require.NoError(t, err)
			state := NewEditorState(100, 100, config.RuleSet{})
			state.documentBuffer.textTree = textTree
			state.documentBuffer.cursor = tc.initialCursor
			mutator := NewDeleteLinesMutator(tc.targetLineLocator, tc.abortIfTargetIsCurrentLine)
			mutator.Mutate(state)
			assert.Equal(t, tc.expectedCursor, state.documentBuffer.cursor)
			assert.Equal(t, tc.expectedText, textTree.String())
			assert.Equal(t, tc.expectedUnsavedChanges, state.hasUnsavedChanges)
		})
	}
}

func TestScrollLinesMutator(t *testing.T) {
	testCases := []struct {
		name               string
		inputString        string
		initialView        viewState
		direction          text.ReadDirection
		numLines           uint64
		expectedtextOrigin uint64
	}{
		{
			name:               "empty, scroll up",
			inputString:        "",
			initialView:        viewState{textOrigin: 0, height: 100, width: 100},
			direction:          text.ReadDirectionBackward,
			numLines:           1,
			expectedtextOrigin: 0,
		},
		{
			name:               "empty, scroll down",
			inputString:        "",
			initialView:        viewState{textOrigin: 0, height: 100, width: 100},
			direction:          text.ReadDirectionForward,
			numLines:           1,
			expectedtextOrigin: 0,
		},
		{
			name:               "scroll up",
			inputString:        "ab\ncd\nef\ngh\nij\nkl\nmn",
			initialView:        viewState{textOrigin: 12, height: 2, width: 100},
			direction:          text.ReadDirectionBackward,
			numLines:           3,
			expectedtextOrigin: 3,
		},
		{
			name:               "scroll down",
			inputString:        "ab\ncd\nef\ngh\nij\nkl\nmn",
			initialView:        viewState{textOrigin: 3, height: 2, width: 100},
			direction:          text.ReadDirectionForward,
			numLines:           3,
			expectedtextOrigin: 12,
		},
		{
			name:               "scroll down to last line",
			inputString:        "ab\ncd\nef\ngh\nij\nkl\nmn",
			initialView:        viewState{textOrigin: 0, height: 6, width: 100},
			numLines:           10,
			expectedtextOrigin: 12,
		},
		{
			name:               "scroll down view taller than document",
			inputString:        "ab\ncd\nef\ngh",
			initialView:        viewState{textOrigin: 0, height: 100, width: 100},
			numLines:           1,
			expectedtextOrigin: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			textTree, err := text.NewTreeFromString(tc.inputString)
			require.NoError(t, err)
			state := NewEditorState(100, 100, config.RuleSet{})
			state.documentBuffer.textTree = textTree
			state.documentBuffer.view = tc.initialView
			mutator := NewScrollLinesMutator(tc.direction, tc.numLines)
			mutator.Mutate(state)
			assert.Equal(t, tc.expectedtextOrigin, state.documentBuffer.view.textOrigin)
		})
	}
}

func TestShowMenuMutator(t *testing.T) {
	state := NewEditorState(100, 100, config.RuleSet{})
	prompt := "test prompt"
	mutator := NewShowMenuMutatorWithItems(prompt, []MenuItem{
		{Name: "test item 1"},
		{Name: "test item 2"},
	}, false)
	mutator.Mutate(state)
	assert.True(t, state.Menu().Visible())
	assert.Equal(t, prompt, state.Menu().Prompt())
	assert.Equal(t, "", state.Menu().SearchQuery())

	results, selectedIdx := state.Menu().SearchResults()
	assert.Equal(t, 0, selectedIdx)
	assert.Equal(t, 0, len(results))
}

func TestHideMenuMutator(t *testing.T) {
	state := NewEditorState(100, 100, config.RuleSet{})
	mutator := NewCompositeMutator([]Mutator{
		NewShowMenuMutatorWithItems("test prompt", []MenuItem{{Name: "test item"}}, false),
		NewHideMenuMutator(),
	})
	mutator.Mutate(state)
	assert.False(t, state.Menu().Visible())
}

func TestSelectAndExecuteMenuItem(t *testing.T) {
	state := NewEditorState(100, 100, config.RuleSet{})
	items := []MenuItem{
		{
			Name:   "set syntax json",
			Action: NewSetSyntaxMutator(syntax.LanguageJson),
		},
		{
			Name:   "quit",
			Action: NewQuitMutator(),
		},
	}
	mutator := NewCompositeMutator([]Mutator{
		NewShowMenuMutatorWithItems("test prompt", items, false),
		NewAppendMenuSearchMutator('q'), // search for "q", should match "quit"
		NewExecuteSelectedMenuItemMutator(),
	})
	mutator.Mutate(state)
	assert.False(t, state.Menu().Visible())
	assert.Equal(t, "", state.Menu().SearchQuery())
	assert.True(t, state.QuitFlag())
}

func TestMoveMenuSelectionMutator(t *testing.T) {
	testCases := []struct {
		name              string
		items             []MenuItem
		searchRune        rune
		moveDeltas        []int
		expectSelectedIdx int
	}{
		{
			name:              "empty results, move up",
			items:             []MenuItem{},
			searchRune:        't',
			moveDeltas:        []int{-1},
			expectSelectedIdx: 0,
		},
		{
			name:              "empty results, move down",
			items:             []MenuItem{},
			searchRune:        't',
			moveDeltas:        []int{1},
			expectSelectedIdx: 0,
		},
		{
			name: "single result, move up",
			items: []MenuItem{
				{Name: "test"},
			},
			searchRune:        't',
			moveDeltas:        []int{1},
			expectSelectedIdx: 0,
		},
		{
			name: "single result, move down",
			items: []MenuItem{
				{Name: "test"},
			},
			searchRune:        't',
			moveDeltas:        []int{1},
			expectSelectedIdx: 0,
		},
		{
			name: "multiple results, move down and up",
			items: []MenuItem{
				{Name: "test1"},
				{Name: "test2"},
				{Name: "test3"},
			},
			searchRune:        't',
			moveDeltas:        []int{2, -1},
			expectSelectedIdx: 1,
		},
		{
			name: "multiple results, move up and wraparound",
			items: []MenuItem{
				{Name: "test1"},
				{Name: "test2"},
				{Name: "test3"},
				{Name: "test4"},
			},
			searchRune:        't',
			moveDeltas:        []int{-1},
			expectSelectedIdx: 3,
		},
		{
			name: "multiple results, move down and wraparound",
			items: []MenuItem{
				{Name: "test1"},
				{Name: "test2"},
				{Name: "test3"},
				{Name: "test4"},
			},
			searchRune:        't',
			moveDeltas:        []int{3, 1},
			expectSelectedIdx: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			state := NewEditorState(100, 100, config.RuleSet{})
			mutators := []Mutator{
				NewShowMenuMutatorWithItems("test", tc.items, false),
				NewAppendMenuSearchMutator(tc.searchRune),
			}
			for _, delta := range tc.moveDeltas {
				mutators = append(mutators, NewMoveMenuSelectionMutator(delta))
			}
			NewCompositeMutator(mutators).Mutate(state)
			_, selectedIdx := state.Menu().SearchResults()
			assert.Equal(t, tc.expectSelectedIdx, selectedIdx)
		})
	}
}

func TestAppendMenuSearchMutator(t *testing.T) {
	state := NewEditorState(100, 100, config.RuleSet{})
	mutator := NewCompositeMutator([]Mutator{
		NewShowMenuMutatorWithItems("test", []MenuItem{}, false),
		NewAppendMenuSearchMutator('a'),
		NewAppendMenuSearchMutator('b'),
		NewAppendMenuSearchMutator('c'),
	})
	mutator.Mutate(state)
	assert.Equal(t, "abc", state.Menu().SearchQuery())
}

func TestDeleteMenuSearchMutator(t *testing.T) {
	testCases := []struct {
		name        string
		searchQuery string
		numDeleted  int
		expectQuery string
	}{
		{
			name:        "delete from empty query",
			searchQuery: "",
			numDeleted:  1,
			expectQuery: "",
		},
		{
			name:        "delete ascii from end of query",
			searchQuery: "abc",
			numDeleted:  2,
			expectQuery: "a",
		},
		{
			name:        "delete non-ascii unicode from end of query",
			searchQuery: "£፴",
			numDeleted:  1,
			expectQuery: "£",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			state := NewEditorState(100, 100, config.RuleSet{})
			mutators := []Mutator{
				NewShowMenuMutatorWithItems("test", []MenuItem{}, false),
			}
			for _, r := range tc.searchQuery {
				mutators = append(mutators, NewAppendMenuSearchMutator(r))
			}
			for i := 0; i < tc.numDeleted; i++ {
				mutators = append(mutators, NewDeleteMenuSearchMutator())
			}
			NewCompositeMutator(mutators).Mutate(state)
			assert.Equal(t, tc.expectQuery, state.Menu().SearchQuery())
		})
	}
}

func TestQuitMutator(t *testing.T) {
	testCases := []struct {
		name              string
		force             bool
		hasUnsavedChanges bool
		expectQuitFlag    bool
	}{
		{
			name:           "no force, no unsaved changes",
			expectQuitFlag: true,
		},
		{
			name:           "force, no unsaved changes",
			force:          true,
			expectQuitFlag: true,
		},
		{
			name:              "no force, unsaved changes",
			hasUnsavedChanges: true,
			expectQuitFlag:    false,
		},
		{
			name:              "force, unsaved changes",
			force:             true,
			hasUnsavedChanges: true,
			expectQuitFlag:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			state := NewEditorState(100, 100, config.RuleSet{})
			state.hasUnsavedChanges = tc.hasUnsavedChanges

			mutator := NewQuitMutator()
			if !tc.force {
				mutator = NewAbortIfUnsavedChangesMutator(mutator, true)
			}

			mutator.Mutate(state)
			assert.Equal(t, tc.expectQuitFlag, state.QuitFlag())
			if !tc.expectQuitFlag {
				assert.Equal(t, StatusMsgStyleError, state.statusMsg.Style)
				assert.Contains(t, state.statusMsg.Text, "Document has unsaved changes")
			}
		})
	}
}