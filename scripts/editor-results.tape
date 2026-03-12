# editor-results.tape - Screenshot of default Results preview mode
# Regenerate: vhs scripts/editor-results.tape
Output site/static/images/editor-results.png

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

Screenshot site/static/images/editor-results.png
