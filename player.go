package klab

import (
	"errors"
	"strings"

	"golang.org/x/net/websocket"
)

type PlayerID int

type Player struct {
	conn *websocket.Conn
	name string
}

func NewPlayer(conn *websocket.Conn, name string) (*Player, error) {
	if conn == nil {
		return nil, errors.New("invalid connection")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("please enter your name")
	}

	return &Player{conn, name}, nil
}
