// Command server runs the authoritative fixed-tick loop around sim.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"exodust/server/sim"
)

const tickRate = 2 // ticks per second

func main() {
	secret, err := loadSecret()
	if err != nil {
		log.Fatal(err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	s := &server{
		secret:  secret,
		origins: strings.Split(os.Getenv("ALLOWED_ORIGINS"), ","),
		inputs:  make(chan sim.Input, 1024),
		join:    make(chan *client),
		leave:   make(chan *client),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	go func() {
		log.Printf("listening on :%s", port)
		log.Fatal(http.ListenAndServe(":"+port, mux))
	}()

	s.run()
}

// run is the only goroutine that touches state and the client list.
func (s *server) run() {
	state := sim.State{}
	clients := map[*client]bool{}
	var inputs []sim.Input

	ticker := time.NewTicker(time.Second / tickRate)
	defer ticker.Stop()

	for {
		select {
		case c := <-s.join:
			clients[c] = true
			log.Printf("player %s connected (%d online)", c.userID, len(clients))

		case c := <-s.leave:
			delete(clients, c)
			close(c.send) // only the sender closes a channel
			log.Printf("player %s left (%d online)", c.userID, len(clients))

		case <-ticker.C:
			var lines []string
			if state.Counter%30 == 0 {
				lines = append(lines, fmt.Sprintf("world state %d", state.Counter))
			}

			// Drain everything queued since the last tick, without waiting.
			for drained := false; !drained; {
				select {
				case in := <-s.inputs:
					inputs = append(inputs, in)
				default:
					drained = true
				}
			}
			state = sim.Tick(state, inputs)
			inputs = inputs[:0]

			lines = append(lines, fmt.Sprintf("state %d %d", state.PlayerX, state.PlayerY))
			for _, line := range lines {
				fmt.Println(line)
				for c := range clients {
					select {
					case c.send <- line:
					default: // slow client: drop the line instead of freezing the world
					}
				}
			}
		}
	}
}
