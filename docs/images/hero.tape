# hero.tape - CalcMark hero demo GIF
# Regenerate: vhs docs/images/hero.tape
# Shows cm eval with a budget calculation featuring variables, expressions, and currency formatting.
Output docs/images/hero.gif

Set Shell "bash"
Set FontSize 18
Set Width 900
Set Height 500
Set Theme "Catppuccin Mocha"
Set Padding 15
Set TypingSpeed 50ms
Set Framerate 12

Require cm

# Create a demo file with calculations
Hide
Type `cat << 'EOF' > /tmp/cm-hero-demo.cm`
Enter
Sleep 100ms
Type "# Monthly Budget"
Enter
Type ""
Enter
Type "income = $5000"
Enter
Type "rent = $1500"
Enter
Type "groceries = $400"
Enter
Type "savings = income * 0.20"
Enter
Type "remaining = income - rent - groceries - savings"
Enter
Type "EOF"
Enter
Sleep 200ms
Show

# Evaluate the file
Type "cm eval /tmp/cm-hero-demo.cm"
Enter
Sleep 3s

# Clean up
Hide
Type "rm /tmp/cm-hero-demo.cm"
Enter
Show
Sleep 500ms
