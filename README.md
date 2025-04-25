# Text Editor

## Progress

### 25/04/2025

Major refactoring day. Moved the core editor code into a `core` package and kept the UI or view in a separate `output` package. The bridge between the two would be the `EditorState` struct that contains the necessary information/functions to update the view.

Now, my output has simple cursor highlight and will be redrawn on editor updates.

I would like to improve the view by adding a status bar next.

### 22/04/2025

So many changes since the last update. It all started out with simply hiding the terminal cursor. But I encountered a problem with my cursor not showing after the entire program exits. Turns out, my deferred function to show the cursor was not being called.

Solving the issue involved refactoring my code and gave me a chance to separate both the output and input. While learning how to design a better code for the setup and teardown of the input listener, I stumbled upon this library [`https://github.com/atomicgo/keyboard`](https://github.com/atomicgo/keyboard) and learned how to handle keypresses better -- also supports utf8!

But now my text editor works and has a decent view. I would like to extract my the "view" code into a separate component, but I'm still thinking on what's the best way to do that. My long term goal is to support multiple views e.g. browser, terminal, etc. An API to fetch the data to populate the view should be a good start.

### 12/04/2025

Had a busy week and was absent for a while. But finally finished implementing undo delete. Made some refactorings and improvements to the code. However, I'm very sure that there are still bugs. The next immediate step is to add tests. Then, probably work on improving the "UI" of the editor as well as better error handling.

Afterwards, I would have a pretty basic text editor. Improvements to it would include a more efficient buffer, text formatting support, UTF8 support.

### 03/04/2025

Updated the undo functionality to delete the last typed word instead of a single character. I am currently tracking the changes using a `changeStart` and `changeLen` variable. Undoes are stored whenever there's a new whitespace or newline. Recalculating the cursor and indices was really difficult and I'm pretty sure my code is still imperfect (there's probably an edge case that I'm not accounting for).

In the process of implementing, I made huge changes to the code and fixed a lot of bugs (largely index errors). I'm sure I will catch more bugs when I try to implement undo delete.

Also, I'm not sure if I will be satisfied to implement my undo as a stack. I looked at some references like vim and other text editors (and plugins), and the action history is stored as a tree. Also, the undo chain is "append only" -- every undo action just creates an inverse of the last action and appends to the history, rather than reversing the last action. While I understand the benefits, I decided to just keep it simple for now.

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
