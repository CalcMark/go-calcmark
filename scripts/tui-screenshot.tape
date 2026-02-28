# tui-screenshot.tape - Capture a PNG screenshot of the TUI editor
# Regenerate: vhs scripts/tui-screenshot.tape
Output site/static/images/tui-screenshot.png

Set Shell "bash"
Set FontSize 16
Set Width 900
Set Height 500
Set Theme "Catppuccin Mocha"
Set Padding 15

Require ./cm

# Open the budget example file
Hide
Type "./cm testdata/examples/budget.cm"
Enter
Show

# Wait for the TUI to fully render
Sleep 3s

# Take the screenshot
Screenshot site/static/images/tui-screenshot.png
