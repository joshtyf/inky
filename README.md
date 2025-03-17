# Text Editor

## Progress

### 18/03/2025

Created a simple write only text editor using a gap buffer. Though the buffer implementation was easy, it was made difficult when incorporating the cursor. My initial design was to store the cursor within the buffer, but that led to a lot of mental math gymnastics. Separating the cursor position into a separate struct, and having the buffer itself recalculate the buffer position based on the cursor input made life a lot easier.

### 12/03/2025

Created the project and implemented a simple text **reader**. The text is read from a file and displayed in the console.
Left and right arrow keys are used to navigate through the text, with each key press moving the cursor by one character and printing from the new position to the next newline character.

Biggest challenge was learning how to read and process each character immediately from `stdin`. I had to disable the default buffering of the console input and read the input character by character. Special characters like the arrow keys had to be handled separately.
