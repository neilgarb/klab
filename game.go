package klab

import (
	"errors"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

type Game struct {
	code        string
	playerCount int

	mu      sync.Mutex
	players []*Player
	started bool
}

func NewGame(code string, playerCount int) (*Game, error) {
	code = strings.TrimSpace(strings.ToUpper(code))
	if code == "" {
		return nil, errors.New("no game code provided")
	}

	if playerCount < 2 || playerCount > 4 {
		return nil, errors.New("number of players should be 2, 3 or 4")
	}

	return &Game{
		code:        code,
		playerCount: playerCount,
	}, nil
}

func (g *Game) PlayerCount() int {
	return g.playerCount
}

func (g *Game) Join(conn *websocket.Conn, name string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if conn == nil {
		return errors.New("invalid connection")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("please enter your name")
	}

	var playerNames []string
	for _, p := range g.players {
		if p.conn == conn {
			return errors.New("you're already in this game")
		}
		if p.name == name {
			return errors.New("that name is already taken")
		}
		playerNames = append(playerNames, p.name)
	}

	if len(g.players) == g.playerCount {
		return errors.New("this game is full")
	}

	player, err := NewPlayer(conn, name)
	if err != nil {
		return err
	}
	g.players = append(g.players, player)
	playerNames = append(playerNames, player.name)

	for i, p := range g.players {
		websocket.JSON.Send(p.conn, MakeMessage("game_lobby", GameLobbyMessage{
			Code:        g.code,
			Host:        i == 0,
			CanStart:    g.playerCount == len(g.players),
			PlayerCount: g.playerCount,
			PlayerNames: playerNames,
		}))
	}

	return nil
}

func (g *Game) MaybeLeave(conn *websocket.Conn) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	var res bool
	var newPlayers []*Player
	var playerNames []string
	for _, p := range g.players {
		if conn == p.conn {
			res = true
			continue
		}
		newPlayers = append(newPlayers, p)
		playerNames = append(playerNames, p.name)
	}
	g.players = newPlayers

	for i, p := range g.players {
		websocket.JSON.Send(p.conn, MakeMessage("game_lobby", GameLobbyMessage{
			Code:        g.code,
			Host:        i == 0,
			CanStart:    g.playerCount == len(g.players),
			PlayerCount: g.playerCount,
			PlayerNames: playerNames,
		}))
	}

	return res
}

func (g *Game) MaybeStart(conn *websocket.Conn) (bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	var found bool
	for _, p := range g.players {
		if p.conn == conn {
			found = true
			break
		}
	}
	if !found {
		return false, nil
	}

	if g.started {
		return false, errors.New("game has already started")
	}

	if len(g.players) != g.playerCount {
		return false, errors.New("game doesn't have enough players")
	}

	g.started = true

	var playerNames []string
	for _, p := range g.players {
		playerNames = append(playerNames, p.name)
	}

	for _, p := range g.players {
		websocket.JSON.Send(p.conn, MakeMessage("game_started", GameStartedMessage{
			Name:        p.name,
			PlayerNames: playerNames,
		}))
	}

	go g.run()

	return true, nil
}

func (g *Game) run() {
	g.mu.Lock()
	var playerNames []string
	for _, p := range g.players {
		playerNames = append(playerNames, p.name)
	}
	g.mu.Unlock()

	scores := make(map[string]int)
	var rounds [][]int

	for {
		var round []int
		for _, p := range playerNames {
			round = append(round, scores[p])
		}
		rounds = append(rounds, round)

		g.mu.Lock()
		for _, p := range g.players {
			websocket.JSON.Send(p.conn, MakeMessage("game_scores", GameScoresMessage{
				PlayerNames: playerNames,
				Scores: rounds,
			}))
		}
		g.mu.Unlock()

		time.Sleep(10*time.Second)
	}
}
