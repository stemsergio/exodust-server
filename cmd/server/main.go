// Command server runs the authoritative fixed-tick loop around sim.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"exodust/server/sim"
)

const tickRate = 2 // ticks per second

func main() {
	state := sim.State{}

	ticker := time.NewTicker(time.Second / tickRate)
	//scanner := bufio.NewScanner(os.Stdin)//no cause it blocks ticks
	defer ticker.Stop()

	inputChan := make(chan sim.Input, 1024)
	inputs := []sim.Input{}

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			for _, key := range strings.Fields(scanner.Text()) {
				inputChan <- sim.Input{KInput: key}
			}
		}
	}()

	// Only this goroutine touches state. Wall-clock time lives here, never in sim.
	for range ticker.C {
		if state.Counter%30 == 0 {
			fmt.Println("world state", state.Counter)
		}
		// Drain everything queued since the last tick, without waiting.
		for drained := false; !drained; {
			select {
			case in := <-inputChan:
				inputs = append(inputs, in)
			default:
				drained = true
			}
		}

		state = sim.Tick(state, inputs)
		fmt.Println("state", state.PlayerX, state.PlayerY)
		// Reuse the slice's backing array on the next tick.
		inputs = inputs[:0]
	}
}
