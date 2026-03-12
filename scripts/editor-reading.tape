# editor-reading.tape - Screenshot of Reading Mode preview
# Regenerate: vhs scripts/editor-reading.tape
Output site/static/images/editor-reading.png

Set Shell "bash"
Set FontSize 16
Set Width 900
Set Height 560
Set Theme "Catppuccin Mocha"
Set Padding 15

Require ./cm

# Open the recipe scaling example (has interpolation + scaling)
Hide
Type "./cm testdata/examples/recipe-scaling.cm"
Enter
Show
Sleep 3s

# Cycle to Reading Mode (Results -> Side-by-Side -> Reading Mode)
Ctrl+P
Sleep 500ms
Ctrl+P
Sleep 1s

Screenshot site/static/images/editor-reading.png
