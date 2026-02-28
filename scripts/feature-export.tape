# feature-export.tape - Shows export format overlay
# Regenerate: vhs scripts/feature-export.tape
# For the "Export anywhere" feature card on the homepage.
Output site/static/images/feature-export.gif

Set Shell "bash"
Set FontSize 16
Set Width 700
Set Height 400
Set Theme "Catppuccin Mocha"
Set Padding 10
Set TypingSpeed 80ms
Set Framerate 12

Require ./cm

# Launch editor with no file
Hide
Type "./cm"
Enter
Show

# Wait for the TUI editor to render
Sleep 2s

# Type some content first
Type "price = $42.50"
Enter
Sleep 300ms
Type "tax = price * 0.08"
Enter
Sleep 300ms
Type "total = price + tax"
Sleep 1s

# Open export overlay
Ctrl+E
Sleep 2s

# Dismiss and exit
Escape
Sleep 500ms

Ctrl+C
