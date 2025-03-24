# Text Editor

## Progress

### 24/03/2025

Mainly did refactorings and bug fixes. I'm refactoring to ensure that I can implement different types of buffers (rope and piece table will likely be next). I'm also looking to implement undo/redo functionality.

### 19/03/2025

Implemented delete functionality. I learned about delete being a special character denoted by the value `127` in ASCII. It also led me on a deeper dive on ASCII and Unicode as I realised the implementation of delete requires the system to know if it's dealing with a single byte character or a multi-byte character.

For now, I will implement my text editor to only handle ASCII/UTF-8 to keep things simple. I will revisit this in the future to handle multi-byte characters.

### 18/03/2025

Created a simple write only text editor using a gap buffer. Though the buffer implementation was easy, it was made difficult when incorporating the cursor. My initial design was to store the cursor within the buffer, but that led to a lot of mental math gymnastics. Separating the cursor position into a separate struct, and having the buffer itself recalculate the buffer position based on the cursor input made life a lot easier.

### 12/03/2025

Created the project and implemented a simple text **reader**. The text is read from a file and displayed in the console.
Left and right arrow keys are used to navigate through the text, with each key press moving the cursor by one character and printing from the new position to the next newline character.

Biggest challenge was learning how to read and process each character immediately from `stdin`. I had to disable the default buffering of the console input and read the input character by character. Special characters like the arrow keys had to be handled separately.
