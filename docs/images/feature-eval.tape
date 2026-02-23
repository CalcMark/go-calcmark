# feature-eval.tape - Shows cm eval with a budget calculation
# Regenerate: vhs docs/images/feature-eval.tape
# For the "Terminal-native" feature card on the homepage.
Output docs/images/feature-eval.gif

Set Shell "bash"
Set FontSize 16
Set Width 700
Set Height 400
Set Theme "Catppuccin Mocha"
Set Padding 10
Set TypingSpeed 50ms
Set Framerate 12

Require cm

# Use printf to pipe expressions into cm eval (single visible command)
Type `printf 'income = $5000\nrent = $1500\ngroceries = $400\nsavings = income * 0.20\nremaining = income - rent - groceries - savings\n' | cm eval`
Enter
Sleep 3s
