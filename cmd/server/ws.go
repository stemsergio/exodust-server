package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"

	"exodust/server/sim"
)

// client is one connected browser. Only the loop goroutine sends into send and closes it.
type client struct {
	userID string
	send   chan string
}

// server holds what every connection needs to reach the loop.
type server struct {
	secret  []byte
	origins []string
	inputs  chan sim.Input
	join    chan *client
	leave   chan *client
}

func (s *server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: s.origins})
	if err != nil {
		return // Accept already answered with an HTTP error
	}
	defer conn.CloseNow()
	conn.SetReadLimit(1024)

	// 1. The first message must be the token, within 5 seconds.
	authCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	_, data, err := conn.Read(authCtx)
	cancel()
	if err != nil {
		return
	}
	userID, err := verifyToken(string(data), s.secret)
	if err != nil {
		log.Printf("rejected connection: %v", err)
		conn.Close(websocket.StatusPolicyViolation, "invalid token")
		return
	}

	c := &client{userID: userID, send: make(chan string, 16)}
	s.join <- c
	defer func() { s.leave <- c }()

	// 2. Writer goroutine: loop -> browser.
	go func() {
		for line := range c.send {
			if err := conn.Write(r.Context(), websocket.MessageText, []byte(line)); err != nil {
				return
			}
		}
	}()

	// 3. Reader (this goroutine): browser -> inputs. Same job the keyboard goroutine did.
	for {
		_, data, err := conn.Read(r.Context())
		if err != nil {
			return
		}
		for _, key := range strings.Fields(string(data)) {
			s.inputs <- sim.Input{KInput: key}
		}
	}
}
