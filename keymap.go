package main

var DEFAULT_KEYMAP = NewBasicKeyMap()

type KeyMap interface {
	mapByte(b byte, charOut chan<- byte, keyOut chan<- SpecialKey) error
	mapSpecialKey(k SpecialKey, charOut chan<- byte, keyOut chan<- SpecialKey) error
}

type BasicKeyMap struct{}

func NewBasicKeyMap() *BasicKeyMap {
	return &BasicKeyMap{}
}

func (km *BasicKeyMap) mapByte(b byte, charOut chan<- byte, keyOut chan<- SpecialKey) error {
	charOut <- b
	return nil
}

func (km *BasicKeyMap) mapSpecialKey(k SpecialKey, charOut chan<- byte, keyOut chan<- SpecialKey) error {
	keyOut <- k
	return nil
}
