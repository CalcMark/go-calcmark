# editor-sidebyside.tape - Screenshot of Side-by-Side preview mode
# Regenerate: vhs scripts/editor-sidebyside.tape
Output site/static/images/editor-sidebyside.png

Set Shell "bash"
Set FontSize 16
Set Width 900
Set Height 560
Set Theme "Catppuccin Mocha"
Set Padding 15

Require ./cm

# Open the recipe scaling example
Hide
Type "./cm testdata/examples/recipe-scaling.cm"
Enter
Show
Sleep 3s

# Cycle to Side-by-Side preview (Results -> Side-by-Side)
Ctrl+P
Sleep 1s

Screenshot site/static/images/editor-sidebyside.png
