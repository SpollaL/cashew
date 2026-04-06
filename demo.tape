# cashew demo
Output docs/demo.gif

Set Shell bash
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 80ms
Set Padding 20

# Launch
Type "./cashew demo_data.csv"
Enter
Sleep 2.5s

# Start in summary view
Type "s"
Sleep 1s

# Cycle granularities
Type "a"
Sleep 1.5s
Type "y"
Sleep 1.5s
Type "M"
Sleep 2s

# Categories pivot
Type "c"
Sleep 1.5s
Type "M"
Sleep 2.5s

# Transactions list — scroll through a few rows
Type "t"
Sleep 2s
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 1.5s

# Review queue — categorize an item
Type "r"
Sleep 2s

# Press enter to open the category picker for the first item
Enter
Sleep 1s

# Navigate down to "Dining" (index 1 in the bucket list)
Down
Sleep 500ms

# Confirm
Enter
Sleep 2s

# Back to summary
Type "s"
Sleep 1s

# Quit
Type "q"
Sleep 500ms
