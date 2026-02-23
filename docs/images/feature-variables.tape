# feature-variables.tape - Shows TUI live preview updating as variables are typed
# Regenerate: vhs docs/images/feature-variables.tape
# For the "Variables flow downward" feature card on the homepage.
Output docs/images/feature-variables.gif

Set Shell "bash"
Set FontSize 16
Set Width 700
Set Height 350
Set Theme "Catppuccin Mocha"
Set Padding 10
Set TypingSpeed 80ms
Set Framerate 12

Require cm

# Launch editor with no file (opens blank document)
Hide
Type "cm edit"
Enter
Show

# Wait for the TUI editor to render (status bar shows Ctrl shortcuts)
Sleep 2s

Type "income = $5000"
Enter
Sleep 800ms

Type "rent = income * 0.30"
Enter
Sleep 800ms

Type "remaining = income - rent"
Sleep 1500ms

Ctrl+C
