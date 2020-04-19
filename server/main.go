package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
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

var addr = flag.String("addr", ":8080", "Listen address")
var debug = flag.String("debug", ":8082", "Debug address")
var staticBase = flag.String("static_base", "", "Static base URL")

func main() {
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	go func() {
		r := httprouter.New()
		r.HandlerFunc("GET", "/debug/pprof/profile", pprof.Profile)
		r.HandlerFunc("GET", "/debug/pprof/symbol", pprof.Symbol)
		r.HandlerFunc("GET", "/debug/pprof/", pprof.Index)
		r.HandlerFunc("GET", "/debug/pprof/block", pprof.Index)
		r.HandlerFunc("GET", "/debug/pprof/goroutine", pprof.Index)
		r.HandlerFunc("GET", "/debug/pprof/heap", pprof.Index)
		r.HandlerFunc("GET", "/debug/pprof/threadcreate", pprof.Index)
		http.ListenAndServe(*debug, r)
	}()

	r := httprouter.New()

	r.GET("/ws", websocketHandler)
	r.GET("/", homeHandler)

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.Dir(path.Join(wd, "/client")))
	r.GET("/client/*filepath", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		cacheUntil := time.Now().AddDate(0, 0, 7).Format(http.TimeFormat)
		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Set("Cache-Control", "public, max-age=7776000")
		w.Header().Set("Expires", cacheUntil)
		r.URL.Path = p.ByName("filepath")
		fileServer.ServeHTTP(w, r)
	})

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

func csp() string {
	return fmt.Sprintf("default-src 'self'; "+
		"img-src 'self' %[1]s; "+
		"media-src 'self' %[1]s; "+
		"script-src 'self' 'unsafe-inline' %[1]s; "+
		"connect-src 'self' %[1]s; "+
		"font-src 'self' https://fonts.gstatic.com; "+
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com %[1]s;",
		*staticBase)
}

const homeT = `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <link rel="stylesheet" href="{{.StaticBase}}/client/klab.css">
  <meta name="viewport" content="width=860,user-scalable=no">
  <title>Jassus, boet!</title>
</head>
<body>
  <div id="klab"></div>
  <div id="overlay" style="display: none">Connecting...</div>
  <div id="error" style="display: none"></div>
  <div id="round_scores" style="display: none"></div>
  <div id="game_scores" style="display: none"></div>
  <script src="{{.StaticBase}}/client/jquery-3.4.1.min.js"></script>
  <script src="{{.StaticBase}}/client/klab.js"></script>
  <script>init({{.StaticBase}});</script>
</body>
</html>`

func homeHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Security-Policy", csp())
	template.Must(template.New("home").Parse(homeT)).Execute(w, struct{
		StaticBase string
	}{*staticBase})
}
