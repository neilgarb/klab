package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
	"html/template"

	"github.com/julienschmidt/httprouter"
	"github.com/neilgarb/klab"
	"golang.org/x/net/websocket"
)

var addr = flag.String("addr", ":8080", "Listen address")

func main() {
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	r := httprouter.New()

	r.GET("/ws", websocketHandler)
	r.GET("/", homeHandler)

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	r.ServeFiles("/client/*filepath", http.Dir(path.Join(wd, "/client")))

	log.Println("Listening")
	http.ListenAndServe(*addr, r)
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

var csp = strings.Join([]string{
	"default-src 'self'; ",
	"img-src 'self'; ",
	"media-src 'self'; ",
	"script-src 'self'; ",
	"connect-src 'self'; ",
	"font-src 'self' https://fonts.gstatic.com; ",
	"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com;",
}, "")

const homeT = `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <link rel="stylesheet" href="client/klab.css">
  <meta name="viewport" content="width=800,user-scalable=no">
  <title>Jassus, boet!</title>
</head>
<body>
  <div id="klab"></div>
  <div id="overlay" style="display: none">Connecting...</div>
  <div id="error" style="display: none"></div>
  <div id="round_scores" style="display: none"></div>
  <div id="game_scores" style="display: none"></div>
  <script src="client/jquery-3.4.1.min.js"></script>
  <script src="client/klab.js"></script>
</body>
</html>`

func homeHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Security-Policy", csp)
	template.Must(template.New("home").Parse(homeT)).Execute(w, nil)
}