package klab

import (
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

type ConnMessage struct {
	Conn    *websocket.Conn
	Message *Message
}

type Game struct {
	code        string
	playerCount int

	mu      sync.Mutex
	players []*Player
	started bool
	ch      chan *ConnMessage
	errCh   chan error
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

	g.ch = make(chan *ConnMessage)
	g.errCh = make(chan error)
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

	dealer := rand.Intn(g.playerCount)

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
				Scores:      rounds,
			}))
		}
		g.mu.Unlock()

		time.Sleep(5 * time.Second)

		g.mu.Lock()
		for _, p := range g.players {
			websocket.JSON.Send(p.conn, MakeMessage("round_started", RoundStartedMessage{
				Dealer: dealer,
			}))
		}
		g.mu.Unlock()

		time.Sleep(time.Second)

		deck := NewDeck(false)
		deck.Shuffle()

		hands := make(map[string][]Card)
		extra := make(map[string][]Card)

		g.mu.Lock()
		for _, p := range g.players {
			hands[p.name] = append(hands[p.name], deck.Deal(3)...)
		}
		for _, p := range g.players {
			hands[p.name] = append(hands[p.name], deck.Deal(3)...)
		}
		cardUp := deck.Deal(1)[0]
		for _, p := range g.players {
			extra[p.name] = append(extra[p.name], deck.Deal(2)...)
		}
		for _, p := range g.players {
			websocket.JSON.Send(p.conn, MakeMessage("round_dealt", RoundDealtMessage{
				PlayerCount: len(g.players),
				Dealer:      dealer,
				DeckSize:    deck.Size(),
				Cards:       hands[p.name],
				CardUp:      cardUp,
			}))
		}
		g.mu.Unlock()

		time.Sleep(5 * time.Second)

		var trumps Suit
		var round2 bool
		toBid := (dealer + 1) % g.playerCount

	BidLoop:
		for {
			g.mu.Lock()
			bidderConn := g.players[toBid].conn
			g.mu.Unlock()

			websocket.JSON.Send(bidderConn, MakeMessage("bid_request", BidRequestMessage{
				CardUp: cardUp,
				Round2: round2,
				Bimah:  round2 && toBid == dealer,
			}))

			for m := range g.ch {
				if m.Message.Type != "bid" {
					g.errCh <- nil
					continue
				}
				if m.Conn != bidderConn {
					g.errCh <- errors.New("it's not your turn to bid")
					continue
				}

				var bidMessage BidMessage
				if err := json.Unmarshal(m.Message.Data, &bidMessage); err != nil {
					g.errCh <- err
					continue
				}

				if !round2 {
					if bidMessage.Pass {
					} else if Suit(bidMessage.Suit) != cardUp.suit {
						g.errCh <- errors.New("must pass or take on")
						continue
					} else {
						trumps = cardUp.suit
						g.errCh <- nil
						break BidLoop
					}
					if toBid == dealer {
						round2 = true
					}
					toBid = (toBid + 1) % g.playerCount
					g.errCh <- nil
					continue BidLoop
				}

				if toBid != dealer {
					if !bidMessage.Pass {
						if Suit(bidMessage.Suit) == cardUp.suit {
							g.errCh <- errors.New("can't take on in that suit")
							continue
						}

						trumps = Suit(bidMessage.Suit)
						g.errCh <- nil
						break BidLoop
					}

					toBid = (toBid + 1) % g.playerCount
					g.errCh <- nil
					continue BidLoop
				}

				trumps = Suit(bidMessage.Suit)
				g.errCh <- nil
				break BidLoop
			}
		}

		g.mu.Lock()
		for _, p := range g.players {
			websocket.JSON.Send(p.conn, MakeMessage("trumps", TrumpsMessage{
				Trumps:     int(trumps),
				ExtraCards: extra[p.name],
			}))
		}
		g.mu.Unlock()

		for m := range g.ch {
			log.Println(g.code, m)
		}

		time.Sleep(10 * time.Second)
		dealer = (dealer + 1) % g.playerCount
	}
}

func (g *Game) MaybePlay(conn *websocket.Conn, msg *Message) (bool, error) {
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

	g.ch <- &ConnMessage{conn, msg}
	if err := <-g.errCh; err != nil {
		return false, err
	}

	return true, nil
}
