# feature-growth-autocomplete.tape - Shows growth function NL autocomplete
# Regenerate: vhs scripts/feature-growth-autocomplete.tape
# Demonstrates compound/grow/depreciate NL completions in autocomplete.
Output site/static/images/feature-growth-autocomplete.gif

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

Type "principal = $1000"
Enter
Sleep 500ms

Type "rate = 5%"
Enter
Sleep 500ms

# Type 'comp' to trigger autocomplete popup showing compound fn + NL row
Type "comp"
Sleep 1500ms

# Navigate to the NL example row (compound by over) and accept
Down
Tab
Sleep 2000ms

Ctrl+C
