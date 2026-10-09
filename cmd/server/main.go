// Command server runs the authoritative fixed-tick loop around sim.
package main

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"exodust/server/sim"
)

const tickRate = 30 // ticks per second

func main() {
	state := sim.State{}

	ticker := time.NewTicker(time.Second / tickRate)
	scanner := bufio.NewScanner(os.Stdin)
	defer ticker.Stop()

	// Only this goroutine touches state. Wall-clock time lives here, never in sim.
	for range ticker.C {
		if state.Counter%30 == 0 {
			fmt.Println("world state", state.Counter)
		}

		state = sim.Tick(state, inputs)
	}
}
