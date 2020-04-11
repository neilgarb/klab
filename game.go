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

		// time.Sleep(5 * time.Second)
		time.Sleep(time.Second)

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

						g.mu.Lock()
						for _, p := range g.players {
							if p.conn == bidderConn {
								continue
							}
							websocket.JSON.Send(p.conn, MakeMessage("speech", SpeechMessage{
								Player:  toBid,
								Message: "Play",
							}))
						}
						g.mu.Unlock()

						break BidLoop
					}
					if toBid == dealer {
						round2 = true
					}
					g.errCh <- nil

					g.mu.Lock()
					for _, p := range g.players {
						if p.conn == bidderConn {
							continue
						}
						websocket.JSON.Send(p.conn, MakeMessage("speech", SpeechMessage{
							Player:  toBid,
							Message: "Pass",
						}))
					}
					g.mu.Unlock()

					toBid = (toBid + 1) % g.playerCount
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

						g.mu.Lock()
						for _, p := range g.players {
							if p.conn == bidderConn {
								continue
							}
							websocket.JSON.Send(p.conn, MakeMessage("speech", SpeechMessage{
								Player:  toBid,
								Message: trumps.String(),
							}))
						}
						g.mu.Unlock()

						break BidLoop
					}

					g.errCh <- nil

					g.mu.Lock()
					for _, p := range g.players {
						if p.conn == bidderConn {
							continue
						}
						websocket.JSON.Send(p.conn, MakeMessage("speech", SpeechMessage{
							Player:  toBid,
							Message: "Pass",
						}))
					}
					g.mu.Unlock()

					toBid = (toBid + 1) % g.playerCount
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

		for k, v := range extra {
			for _, vv := range v {
				hands[k] = append(hands[k], vv)
			}
		}
		extra = nil

		wonCards := make(map[string][]Card)

		toPlay := (dealer + 1) % g.playerCount
		for i := 0; i < 8; i++ {

			trick := make([]Card, 0, g.playerCount)

			for j := 0; j < g.playerCount; j++ {
				g.mu.Lock()
				playerConn := g.players[(toPlay+j)%g.playerCount].conn
				playerName := g.players[(toPlay+j)%g.playerCount].name
				g.mu.Unlock()

				websocket.JSON.Send(playerConn, MakeMessage("your_turn", YourTurnMessage{}))

				var canOverTrump bool
				var maxTrickTrump int
				for _, t := range trick {
					if t.suit != trumps {
						continue
					}
					if t.rank.TrumpRank() > maxTrickTrump {
						maxTrickTrump = t.rank.TrumpRank()
					}
					for _, c := range hands[playerName] {
						if c.suit != trumps {
							continue
						}
						if c.rank.TrumpRank() > t.rank.TrumpRank() {
							canOverTrump = true
						}
					}
				}

				var haveTrumps bool
				for _, c := range hands[playerName] {
					if trumps != SuitUnknown && c.suit == trumps {
						haveTrumps = true
						break
					}
				}

				canPlay := make(map[Card]bool)
				for _, c := range hands[playerName] {
					if len(trick) == 0 {
						canPlay[c] = true
					} else if c.suit == trick[0].suit {
						canPlay[c] = true
					} else if trumps == SuitUnknown {
						canPlay[c] = true
					} else if c.suit == trumps {
						// card must higher than any trumps already played
						if maxTrickTrump > 0 && !canOverTrump {
							canPlay[c] = true
						} else if c.rank.TrumpRank() > maxTrickTrump {
							canPlay[c] = true
						}
					} else {
						if !haveTrumps {
							canPlay[c] = true
						} else if maxTrickTrump > 0 && !canOverTrump {
							canPlay[c] = true
						}
					}
				}

				for m := range g.ch {
					if m.Message.Type != "play" {
						g.errCh <- nil
						continue
					}

					if m.Conn != playerConn {
						g.errCh <- errors.New("it's not your turn")
						continue
					}

					var playMessage PlayMessage
					if err := json.Unmarshal(m.Message.Data, &playMessage); err != nil {
						g.errCh <- err
						continue
					}

					if !canPlay[playMessage.Card] {
						g.errCh <- errors.New("can't play that card")
						continue
					}

					g.errCh <- nil

					trick = append(trick, playMessage.Card)
					newHand := make([]Card, 0, len(hands[playerName])-1)
					for _, c := range hands[playerName] {
						if c == playMessage.Card {
							continue
						}
						newHand = append(newHand, c)
					}
					hands[playerName] = newHand

					g.mu.Lock()
					for _, p := range g.players {
						websocket.JSON.Send(p.conn, MakeMessage("trick", TrickMessage{
							PlayerCount: g.playerCount,
							Dealer:      dealer,
							Cards:       trick,
						}))
					}
					g.mu.Unlock()

					break
				}
			}

			// Everyone's played their cards. Who wins?
			bestPlayer := (dealer + 1) % g.playerCount
			bestCard := trick[0]
			for i := 1; i < len(trick); i++ {
				c := trick[i]
				if c.suit == trumps {
					if bestCard.suit != trumps {
						bestCard = c
						bestPlayer = (dealer + 1 + i) % g.playerCount
						continue
					} else if c.rank.TrumpRank() > bestCard.rank.TrumpRank() {
						bestCard = c
						bestPlayer = (dealer + 1 + i) % g.playerCount
						continue
					}
				}
				if c.suit == bestCard.suit && c.rank > bestCard.rank {
					bestCard = c
					bestPlayer = (dealer + 1 + i) % g.playerCount
					continue
				}
			}

			time.Sleep(2 * time.Second)

			g.mu.Lock()
			wonCards[g.players[bestPlayer].name] = append(
				wonCards[g.players[bestPlayer].name], trick...)
			for _, p := range g.players {
				websocket.JSON.Send(p.conn, MakeMessage("trick_won", TrickWonMessage{
					PlayerCount: g.playerCount,
					Dealer:      dealer,
					Winner:      bestPlayer,
				}))
			}
			g.mu.Unlock()

			trick = nil
		}

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
