package core

import "fmt"

type Operation interface {
	Invert() Operation
	Merge(other Operation) (Operation, bool)
	Apply(e *Editor)
}

type InsertSequence int

const (
	InsertSequenceSingleWhitespace InsertSequence = iota
	InsertSequenceMultiWhitespace
	InsertSequenceSingleNewline
	InsertSequenceCharacterOnly
	InsertSequenceMixed // Mix = A single whitespace or a newline followed by characters
)

type InsertOperation struct {
	Start          int
	Runes          []rune
	InsertSequence InsertSequence
}

func NewInsertOperation(start int, r rune) InsertOperation {
	sequence := InsertSequenceCharacterOnly
	switch r {
	case ' ':
		sequence = InsertSequenceSingleWhitespace
	case '\n':
		sequence = InsertSequenceSingleNewline
	}
	return InsertOperation{
		Start:          start,
		Runes:          []rune{r},
		InsertSequence: sequence,
	}
}

func (op InsertOperation) Invert() Operation {
	seq := DeleteSequenceSingleNonNewline
	if len(op.Runes) == 1 && op.Runes[0] == '\n' {
		seq = DeleteSequenceSingleNewline
	} else if len(op.Runes) > 1 {
		seq = DeleteSequenceMixed
	}
	return DeleteOperation{Start: op.Start, Runes: op.Runes, DeleteSequence: seq}
}

func (op InsertOperation) Merge(other Operation) (Operation, bool) {
	previousInsert, ok := other.(InsertOperation)
	if !ok {
		return nil, false
	}
	// Ensure that the previousInsert cursor position is before the current insert operation cursor
	// Else, swap the operations and attempt to merge again
	if previousInsert.Start > op.Start {
		return previousInsert.Merge(op)
	}
	if len(op.Runes) == 0 || len(previousInsert.Runes) == 0 {
		panic(fmt.Sprintf("Can't merge an insert operation with no runes, op: %+v, otherInsert: %+v", op, previousInsert))
	}

	adjacentInserts := previousInsert.Start+len(previousInsert.Runes) == op.Start
	if !adjacentInserts {
		return nil, false
	}

	previousSequence := previousInsert.InsertSequence
	opSequence := op.InsertSequence
	var mergedSequence InsertSequence
	switch {
	case previousSequence == InsertSequenceSingleWhitespace && opSequence == InsertSequenceCharacterOnly:
		mergedSequence = InsertSequenceMixed
	case previousSequence == InsertSequenceSingleNewline && opSequence == InsertSequenceCharacterOnly:
		mergedSequence = InsertSequenceMixed
	case previousSequence == InsertSequenceSingleWhitespace && opSequence == InsertSequenceSingleWhitespace:
		mergedSequence = InsertSequenceMultiWhitespace
	case previousSequence == InsertSequenceMultiWhitespace && opSequence == InsertSequenceSingleWhitespace:
		mergedSequence = InsertSequenceMultiWhitespace
	case previousSequence == InsertSequenceCharacterOnly && opSequence == InsertSequenceCharacterOnly:
		mergedSequence = InsertSequenceCharacterOnly
	case previousSequence == InsertSequenceMixed && opSequence == InsertSequenceCharacterOnly:
		mergedSequence = InsertSequenceMixed
	default:
		// Cannot be merged
		return nil, false
	}

	mergedRunes := make([]rune, 0, len(op.Runes)+len(previousInsert.Runes))
	mergedRunes = append(mergedRunes, previousInsert.Runes...)
	mergedRunes = append(mergedRunes, op.Runes...)
	return InsertOperation{
		Start:          previousInsert.Start,
		Runes:          mergedRunes,
		InsertSequence: mergedSequence,
	}, true
}

func (op InsertOperation) Apply(e *Editor) {
	e.setCursorPosition(op.Start)
	for _, r := range op.Runes {
		e.insertRune(r)
	}
}

// Sequences should always be read left to right
// For example, if the original word is 'ab' and the user deletes 'b' and then 'a',
// the delete sequence should be read as 'ab'
// This is done to fix a frame of reference for merging deletes
type DeleteSequence int

const (
	DeleteSequenceSingleNewline DeleteSequence = iota
	DeleteSequenceSingleNonNewline
	DeleteSequenceMixed // Mix = All characters, whitespace or newlines. Leftmost character in sequence cannot be a newline (i.e., newline marks a new deletion boundary)
)

type DeleteOperation struct {
	Start          int
	Runes          []rune
	DeleteSequence DeleteSequence
}

func NewDeleteOperation(start int, r rune) DeleteOperation {
	sequence := DeleteSequenceSingleNonNewline
	if r == '\n' {
		sequence = DeleteSequenceSingleNewline
	}
	return DeleteOperation{
		Start:          start,
		Runes:          []rune{r},
		DeleteSequence: sequence,
	}
}

func (op DeleteOperation) Invert() Operation {
	seq := InsertSequenceCharacterOnly
	if len(op.Runes) == 1 {
		switch op.Runes[0] {
		case ' ':
			seq = InsertSequenceSingleWhitespace
		case '\n':
			seq = InsertSequenceSingleNewline
		}
	} else if len(op.Runes) > 1 {
		seq = InsertSequenceMixed
	}
	return InsertOperation{Start: op.Start, Runes: op.Runes, InsertSequence: seq}
}

func (op DeleteOperation) Merge(other Operation) (Operation, bool) {
	previousDelete, ok := other.(DeleteOperation)
	if !ok {
		return nil, false
	}
	// Ensure that the current delete cursor position is before the previous delete cursor
	// Else, swap the operations and attempt to merge again
	if previousDelete.Start < op.Start {
		return previousDelete.Merge(op)
	}
	if len(op.Runes) == 0 || len(previousDelete.Runes) == 0 {
		panic(fmt.Sprintf("Can't merge a delete operation with no runes, op: %+v, previousDelete: %+v", op, previousDelete))
	}
	adjacentDeletes := op.Start+len(op.Runes) == previousDelete.Start
	if !adjacentDeletes {
		return nil, false
	}

	opSequence := op.DeleteSequence
	if opSequence == DeleteSequenceSingleNewline {
		return nil, false
	}

	mergedRunes := make([]rune, 0, len(op.Runes)+len(previousDelete.Runes))
	mergedRunes = append(mergedRunes, op.Runes...)
	mergedRunes = append(mergedRunes, previousDelete.Runes...)
	return DeleteOperation{
		Start:          op.Start,
		Runes:          mergedRunes,
		DeleteSequence: DeleteSequenceMixed,
	}, true
}

func (op DeleteOperation) Apply(e *Editor) {
	e.setCursorPosition(op.Start + len(op.Runes))
	for range op.Runes {
		e.backspace()
	}
}
