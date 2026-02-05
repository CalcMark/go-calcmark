# hero.tape - CalcMark demo GIF
# Run from project root: vhs docs/images/hero.tape
Output docs/images/hero.gif

Set Shell "bash"
Set FontSize 18
Set Width 900
Set Height 500
Set Theme "Catppuccin Mocha"
Set Padding 15
Set TypingSpeed 50ms
Set Framerate 12

# Create a demo file with calculations
Type "cat << 'EOF' > demo.cm"
Enter
Sleep 200ms
Type "# Monthly Budget"
Enter
Type ""
Enter
Type "income = $5000"
Enter
Type "rent = $1500"
Enter
Type "utilities = $200"
Enter
Type "groceries = $400"
Enter
Type ""
Enter
Type "savings = income * 0.20"
Enter
Type "remaining = income - rent - utilities - groceries - savings"
Enter
Type "EOF"
Enter
Sleep 500ms

# Evaluate the file
Type "./cm eval demo.cm"
Enter
Sleep 2s

# Clean up
Type "rm demo.cm"
Enter
Sleep 500ms
