package main

import (
	"flag"
	"math/rand"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/neilgarb/klab"
	"golang.org/x/net/websocket"
)

func main() {
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	r := httprouter.New()

	r.GET("/ws", websocketHandler)

	http.ListenAndServe(":8081", r)
}

func websocketHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	websocket.Handler(newConn).ServeHTTP(w, r)
}

var manager = klab.NewManager()

func newConn(conn *websocket.Conn) {
	for {
		var msg klab.Message
		err := websocket.JSON.Receive(conn, &msg)
		if err != nil {
			conn.Close()
			return
		}

		if err := manager.Handle(conn, &msg); err != nil {
			errMsg := klab.MakeMessage("error", err.Error())
			if err := websocket.JSON.Send(conn, errMsg); err != nil {
				conn.Close()
				return
			}
		}
	}
}
