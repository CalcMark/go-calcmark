# feature-autocomplete.tape - Shows autocomplete popup appearing
# Regenerate: vhs scripts/feature-autocomplete.tape
# For the "Terminal-native" feature card on the homepage.
Output site/static/images/feature-autocomplete.gif

Set Shell "bash"
Set FontSize 16
Set Width 700
Set Height 350
Set Theme "Catppuccin Mocha"
Set Padding 10
Set TypingSpeed 80ms
Set Framerate 12

Require cm

# Launch editor with no file
Hide
Type "cm edit"
Enter
Show

# Wait for the TUI editor to render
Sleep 2s

Type "x = 100"
Enter
Sleep 500ms

Type "y = 200"
Enter
Sleep 500ms

# Type 'av' to trigger autocomplete popup showing avg function
Type "av"
Sleep 1500ms

# Accept autocomplete suggestion
Tab
Type "x, y)"
Sleep 1500ms

Ctrl+C
