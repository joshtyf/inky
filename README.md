# Text Editor

## Progress

### 12/03/2025

Created the project and implemented a simple text **reader**. The text is read from a file and displayed in the console.
Left and right arrow keys are used to navigate through the text, with each key press moving the cursor by one character and printing from the new position to the next newline character.

Biggest challenge was learning how to read and process each character immediately from `stdin`. I had to disable the default buffering of the console input and read the input character by character. Special characters like the arrow keys had to be handled separately.
