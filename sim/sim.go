// Package sim is the game as a pure function: Tick(state) -> state.
// It does no I/O and reads no clock; the caller decides when a tick happens.
package sim

// State is the whole world at one tick.
type State struct {
	Counter int64
	PlayerX int64
	PlayerY int64
}
type Input struct {
	KInput string
}

// Tick advances the world by exactly one tick.
// State is passed and returned by value, so the caller's copy is never mutated.
func Tick(s State, inputs []Input) State {
	s.Counter++

	for _, input := range inputs {
		switch input.KInput {
		case "w":
			s.PlayerY++
		case "s":
			s.PlayerY--
		case "a":
			s.PlayerX--
		case "d":
			s.PlayerX++

		}
	}
	return s
}
