package klab

import (
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"strings"
	"sync"

	"golang.org/x/net/websocket"
)

type Manager struct {
	gamesMu sync.Mutex
	games   map[string]*Game
}

func NewManager() *Manager {
	return &Manager{games: make(map[string]*Game)}
}

func (m *Manager) Handle(conn *websocket.Conn, msg *Message) error {
	switch msg.Type {
	case "create_game":
		var data CreateGameMessage
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return err
		}
		return m.CreateGame(conn, data)
	case "join_game":
		var data JoinGameMessage
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return err
		}
		return m.JoinGame(conn, data)
	case "leave_game":
		return m.LeaveGame(conn)
	case "start_game":
		return m.StartGame(conn)
	case "bid", "play", "announce_bonus":
		return m.Play(conn, msg)
	case "speech":
		var data SpeechMessage
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return err
		}
		return m.Speech(conn, data.Message)
	}
	return errors.New("unknown message type")
}

func (m *Manager) CreateGame(conn *websocket.Conn, msg CreateGameMessage) error {
	m.gamesMu.Lock()
	defer m.gamesMu.Unlock()

	log.Printf("%s: create_game (%+v)", conn.Request().RemoteAddr, msg)

	var code string
	for {
		code = makeGameCode()
		if _, ok := m.games[code]; !ok {
			break
		}
	}

	game, err := NewGame(code)
	if err != nil {
		return err
	}
	m.games[code] = game

	if err := game.Join(conn, msg.Name); err != nil {
		return err
	}

	return nil
}

const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func makeGameCode() string {
	var code string
	for i := 0; i < 4; i++ {
		code += string(alphabet[rand.Intn(len(alphabet))])
	}
	return code
}

func (m *Manager) JoinGame(conn *websocket.Conn, msg JoinGameMessage) error {
	code := strings.TrimSpace(strings.ToUpper(msg.Code))
	if code == "" {
		return errors.New("please enter a game code")
	}

	m.gamesMu.Lock()
	game, ok := m.games[code]
	m.gamesMu.Unlock()

	log.Printf("%s -> %s: %s (%+v)",
		conn.Request().RemoteAddr, game.code, "join_game", msg)

	if !ok {
		return errors.New("game not found")
	}

	name := strings.TrimSpace(msg.Name)
	if name == "" {
		return errors.New("please enter your name")
	}

	if err := game.Join(conn, msg.Name); err != nil {
		return err
	}

	return nil
}

func (m *Manager) LeaveGame(conn *websocket.Conn) error {
	m.gamesMu.Lock()
	defer m.gamesMu.Unlock()

	for _, g := range m.games {
		if g.MaybeLeave(conn) {
			log.Printf("%s -> %s: %s",
				conn.Request().RemoteAddr, g.code, "leave_game")
			break
		}
	}

	return nil
}

func (m *Manager) StartGame(conn *websocket.Conn) error {
	m.gamesMu.Lock()
	defer m.gamesMu.Unlock()

	for _, g := range m.games {
		ok, err := g.MaybeStart(conn)
		if err != nil {
			return err
		}
		if ok {
			log.Printf("%s -> %s: %s",
				conn.Request().RemoteAddr, g.code, "start_game")
			break
		}
	}

	return nil
}

func (m *Manager) Play(conn *websocket.Conn, msg *Message) error {
	m.gamesMu.Lock()
	defer m.gamesMu.Unlock()

	for _, g := range m.games {
		ok, err := g.MaybePlay(conn, msg)
		if err != nil {
			return err
		}
		if ok {
			log.Printf("%s -> %s: %s (%s)",
				conn.Request().RemoteAddr, g.code, msg.Type, string(msg.Data))
			break
		}
	}

	return nil
}

func (m *Manager) Speech(conn *websocket.Conn, message string) error {
	if len(message) > 160 {
		return errors.New("message too long")
	}

	m.gamesMu.Lock()
	defer m.gamesMu.Unlock()

	for _, g := range m.games {
		ok, err := g.MaybeSay(conn, message)
		if err != nil {
			return err
		}
		if ok {
			break
		}
	}

	return nil
}
