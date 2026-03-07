# hero.tape - CalcMark hero demo GIF (TUI editor session)
# Regenerate: task record-demos
# Shows the TUI editor with variables, units, NL functions, and napkin math.
Output site/static/images/hero.gif

Set Shell "bash"
Set FontSize 18
Set Width 900
Set Height 560
Set Theme "Catppuccin Mocha"
Set Padding 15
Set TypingSpeed 55ms
Set Framerate 12

Require ./cm

# Launch editor with no file (opens blank document)
Hide
Type "./cm"
Enter
Show

# Wait for the TUI editor to render
Sleep 2s

# Markdown heading (ignored by evaluator)
Type "# Trip to Berlin"
Enter
Enter
Sleep 600ms

# Variables with $ notation
Type "flight = $1200"
Enter
Sleep 600ms

Type "hotel = 5 * $180"
Enter
Sleep 600ms

Type "food = 7 * $45"
Enter
Sleep 600ms

Type "total = flight + hotel + food"
Enter
Sleep 800ms

Type "15% of total"
Enter
Sleep 1s

# Unit conversions
Enter
Type "## Quick conversions"
Enter
Enter
Sleep 400ms

Type "5 miles in km"
Enter
Enter
Sleep 600ms

Type "20 celsius in fahrenheit"
Enter
Sleep 800ms

# NL functions
Enter
Enter
Type "## Infrastructure check"
Enter
Enter
Sleep 400ms

Type "read 100 MB from ssd"
Enter
Sleep 600ms

Type "compress 1 GB using gzip"
Enter
Sleep 800ms

# Napkin rounding
Type "1500000 as napkin"
Enter
Sleep 800ms
Enter
Sleep 2s

Ctrl+C
