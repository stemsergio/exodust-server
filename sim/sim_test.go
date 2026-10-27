package sim

import "testing"

func TestTickIncrementsCounter(t *testing.T) {
	s := State{}
	for i := int64(1); i <= 3; i++ {
		s = Tick(s, nil)
		if s.Counter != i {
			t.Fatalf("after %d ticks: got Counter=%d, want %d", i, s.Counter, i)
		}
	}
}

func TestTickIsDeterministic(t *testing.T) {
	a, b := State{Counter: 41}, State{Counter: 41}
	if Tick(a, nil) != Tick(b, nil) {
		t.Fatal("same state produced different results")
	}
	if a.Counter != 41 {
		t.Fatal("Tick mutated its input")
	}
}
