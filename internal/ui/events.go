package ui

import "time"

type Msg = any
type Cmd func() Msg

type KeyType int

const (
	KeyRunes KeyType = iota
	KeyControl
)

type KeyMsg struct {
	Name  string
	Type  KeyType
	Runes []rune
	Paste bool
}

func (k KeyMsg) String() string {
	const empty = ""
	if k.Name != empty {
		return k.Name
	}
	return string(k.Runes)
}

type BatchMsg struct{ Commands []Cmd }
type SequenceMsg struct{ Commands []Cmd }
type ClearMsg struct{}
type QuitMsg struct{}

func Batch(commands ...Cmd) Cmd    { return func() Msg { return BatchMsg{Commands: commands} } }
func Sequence(commands ...Cmd) Cmd { return func() Msg { return SequenceMsg{Commands: commands} } }
func ClearScreen() Msg             { return ClearMsg{} }
func Quit() Msg                    { return QuitMsg{} }

func Tick(interval time.Duration, callback func(time.Time) Msg) Cmd {
	return func() Msg {
		timer := time.NewTimer(interval)
		defer timer.Stop()
		return callback(<-timer.C)
	}
}
