# editor-help.tape - Screenshot of help overlay
# Regenerate: vhs scripts/editor-help.tape
Output site/static/images/editor-help.png

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

# Open help overlay
Ctrl+H
Sleep 1s

Screenshot site/static/images/editor-help.png
