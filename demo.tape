Output demo.gif

Set Shell zsh
Set FontSize 14
Set Width 1100
Set Height 560
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms
Set PlaybackSpeed 1.0

Hide
Type "cd /Users/pgavlin/dev/sandbox/tea-grid && clear" Enter
Sleep 400ms
Show

Type "go run ./examples/querybar/" Enter
Sleep 2500ms

# Navigate down the unsorted list.
Down@150ms 5
Sleep 500ms

# Sort by Priority -- focus the column, ascending, then descending.
Right@150ms 2
Sleep 400ms
Type "s"
Sleep 1200ms
Type "s"
Sleep 1500ms

# Filter to open issues.
Type "/"
Sleep 300ms
Type "State:open"
Sleep 600ms
Enter
Sleep 1500ms

# Add a priority floor.
Type "/"
Sleep 300ms
Type " Priority:>3"
Sleep 600ms
Enter
Sleep 1800ms

# Clear filters; sort persists.
Escape
Sleep 1200ms

# Group by State -- focus the column, then toggle grouping.
Left@150ms 1
Sleep 300ms
Type "G"
Sleep 1500ms

# Expand each group in turn.
Type "l"
Sleep 1000ms
Down@180ms 1
Type "l"
Sleep 1000ms
Down@180ms 1
Type "l"
Sleep 3000ms

Hide
Type "q"
