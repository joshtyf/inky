# Text Editor

## Progress

### 30/03/2025

Implemented a very naive undo functionality. I thought I could use a simple command pattern, however my commands are either cursor movements or editing 1 byte at a time. This makes the undo function a very terrible user experience. After playing around with other text editors like Apple Notes, I realised that undo should at least delete the last typed word or if the content is bulk inserted (e.g. pasting), it should delete the entire pasted content. Such a functionality is way harder to implement and I will need to rethink my design. Unfortunately, there is not a lot of information readily available online for me to consult. I did find a few good ones that point me in a general direction, but not enough to implement a good solution. Life's tough.

### 26/03/2025

Implemented a basic keymapping for the text editor. With this keymapping, I can potentially implement more complex keymaps (possibly vim motions?). Toughest part was figuring out how to design my code that adhere to good design principles. I'm also thinking if features like keymapping, undo/redo, etc. should be implemented as middleware -- basically passing the user input through a series of transformation functions before it reaches the buffer. Not sure if this is the right thinking though.

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
