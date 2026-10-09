// Package sim is the game as a pure function: Tick(state) -> state.
// It does no I/O and reads no clock; the caller decides when a tick happens.
package sim

// State is the whole world at one tick.
type State struct {
	Counter int64
	PlayerX uint64
	PlayerY uint64
}
type Input struct {
}

// Tick advances the world by exactly one tick.
// State is passed and returned by value, so the caller's copy is never mutated.
func Tick(s State, inputs []Input) State {
	s.Counter++
	return s
}
