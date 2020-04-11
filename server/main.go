package main

import (
	"flag"
	"math/rand"
	"net/http"
	"net/http/pprof"
	"os"
	"path"
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
	r.HandlerFunc("GET", "/debug/pprof/profile", pprof.Profile)
	r.HandlerFunc("GET", "/debug/pprof/symbol", pprof.Symbol)
	r.HandlerFunc("GET", "/debug/pprof/", pprof.Index)
	r.HandlerFunc("GET", "/debug/pprof/block", pprof.Index)
	r.HandlerFunc("GET", "/debug/pprof/goroutine", pprof.Index)
	r.HandlerFunc("GET", "/debug/pprof/heap", pprof.Index)
	r.HandlerFunc("GET", "/debug/pprof/threadcreate", pprof.Index)

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	r.ServeFiles("/client/*filepath", http.Dir(path.Join(wd, "/client")))

	http.ListenAndServe(":8080", r)
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
