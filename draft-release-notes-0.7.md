# Summary

Release announcement blog post: TODO

* Support count for cursor movements (h/j/k/l and arrow keys).
* Add star (*) and pound (#) commands for forward/backward word search.
* Add workingDir menu command mode.
* Add insertChoice menu command mode.
* Validate that menu command name and shellCmd are nonempty.
* Add gotemplate syntax language.
* Add `[(` and `])` commands to match next/prev parentheses.
* Add `[{` and `]}` commands to match next/prev braces.
* Add % command to find matching brace, bracket, or parenthesis.
* Preserve order of file locations menu items.
* Custom menu command save flag now writes the file only if there are unsaved changes.
* Set $COLUMN env var for shell commands.
* Add menu commands to change the current working directory.
* Support count for word movement and word object commands.
* Remove "set syntax" menu commands.
* Insert text from a shell cmd after the cursor, not before.
* Add FreeBSD as a supported platform and release target.
* Fix markdown syntax highlighting for close code fence with CRLF.
* Fix markdown parsing for code span in emphasis.

# Upgrade Notes

* Aretext now validates that the menu commands in the configuration file have a "shellCmd" key with a non-empty value. If aretext fails on startup with a validation error, follow [these steps to fix it](https://aretext.org/docs/configuration/#fixing-errors-on-startup).

* The "set syntax" menu items have been removed. This avoids some confusing behavior in which the menu commands would use the color palette based on configuration for the current file.
