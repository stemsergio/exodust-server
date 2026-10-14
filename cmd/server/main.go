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

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for _, key := range strings.Fields(scanner.Text()) {
			inputChan <- sim.Input{KInput: key}
		}
	}()

	inputs := []sim.Input{}

	// Only this goroutine touches state. Wall-clock time lives here, never in sim.
	for range ticker.C {
		if state.Counter%30 == 0 {
			fmt.Println("world state", state.Counter)
		}
		//here i need to get the imputs from keyboard in terminal

		state = sim.Tick(state, inputs)
		fmt.Println("state", state.PlayerX, state.PlayerY)
		// Reuse the slice's backing array on the next tick.
		inputs = inputs[:0]
	}
}

/**
package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

func main() {
	fmt.Println("Program started")
	fmt.Println("Type something and press ENTER.")

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			fmt.Printf("Received: %q\n", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			fmt.Println("Input error:", err)
		}
		fmt.Println("Input stream closed")
	}()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var ticks int
	for range ticker.C {
		ticks++
		fmt.Printf("Tick %d\n", ticks)
	}
}
*/
