package state

import (
	"github.com/aretext/aretext/selection"
)

// InputMode controls how the editor interprets input events.
type InputMode int

const (
	InputModeNormal = InputMode(iota)
	InputModeInsert
	InputModeMenu
	InputModeSearch
	InputModeVisual
	InputModeTask
)

func (im InputMode) String() string {
	switch im {
	case InputModeNormal:
		return "normal"
	case InputModeInsert:
		return "insert"
	case InputModeMenu:
		return "menu"
	case InputModeSearch:
		return "search"
	case InputModeVisual:
		return "visual"
	case InputModeTask:
		return "task"
	default:
		panic("invalid input mode")
	}
}

// SetInputMode sets the editor input mode.
func SetInputMode(state *EditorState, mode InputMode) {
	if state.inputMode != mode && mode == InputModeNormal && !state.macroState.isReplayingUserMacro {
		// Transition back to normal mode should set an undo checkpoint.
		// For example, suppose a user adds text in insert mode, then returns to normal mode,
		// then deletes a line.  The next undo should restore the deleted line, returning to
		// the checkpoint AFTER the user changed from insert->normal mode.
		CheckpointUndoLog(state)

	}

	if state.inputMode == InputModeVisual && (mode == InputModeNormal || mode == InputModeInsert) {
		// Clear selection when exiting visual mode.
		state.documentBuffer.selector.Clear()
	}

	state.prevInputMode = state.inputMode
	state.inputMode = mode
}

// ToggleVisualMode transitions to/from visual selection mode.
func ToggleVisualMode(state *EditorState, selectionMode selection.Mode) {
	buffer := state.documentBuffer

	// If we're not already in visual mode, enter visual mode and start a new selection.
	if state.inputMode != InputModeVisual {
		SetInputMode(state, InputModeVisual)
		buffer.selector.Start(selectionMode, buffer.cursor.position)
		return
	}

	// If we're in visual mode but not in the same selection mode,
	// stay in visual mode and change the selection mode
	// (for example, switch from selecting charwise to selecting linewise)
	if buffer.selector.Mode() != selectionMode {
		buffer.selector.SetMode(selectionMode)
		return
	}

	// If we're already in visual mode and the requested selection mode,
	// toggle back to normal mode.  This will also clear the selection.
	SetInputMode(state, InputModeNormal)
}
