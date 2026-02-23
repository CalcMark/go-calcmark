# hero.tape - CalcMark hero demo GIF (TUI editor session)
# Regenerate: vhs docs/images/hero.tape
# Shows the TUI editor with live-updating variables and results pane.
Output docs/images/hero.gif

Set Shell "bash"
Set FontSize 18
Set Width 900
Set Height 500
Set Theme "Catppuccin Mocha"
Set Padding 15
Set TypingSpeed 60ms
Set Framerate 12

Require cm

# Launch editor with no file (opens blank document)
Hide
Type "cm edit"
Enter
Show

# Wait for the TUI editor to render
Sleep 2s

# Build a budget calculation line by line, showing live results
Type "income = $5000"
Enter
Sleep 800ms

Type "rent = $1500"
Enter
Sleep 800ms

Type "groceries = $400"
Enter
Sleep 800ms

Type "savings = income * 0.20"
Enter
Sleep 800ms

Type "remaining = income - rent - groceries - savings"
Sleep 2s

Ctrl+C
