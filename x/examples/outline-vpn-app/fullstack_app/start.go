package fullstack_app

import (
	"io"
	"net/http"

	"github.com/a-h/templ"
	"golang.org/x/net/websocket"
)

func EchoServer(ws *websocket.Conn) {
	io.Copy(ws, ws)
}

func Start() {
	http.Handle("/", templ.Handler(serverView()))
	http.Handle("/__reload__", websocket.Handler(EchoServer))

	http.ListenAndServe(":8080", nil)
}
