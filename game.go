package klab

import (
	"encoding/json"
	"errors"
	"math/rand"
	"sort"
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
		g.mu.Lock()
		total := make([]int, g.playerCount)
		for i, p := range g.players {
			total[i] = scores[p.name]
		}
		for _, p := range g.players {
			websocket.JSON.Send(p.conn, MakeMessage("game_scores", GameScoresMessage{
				PlayerNames: playerNames,
				Total:       total,
				Scores:      rounds,
			}))
		}
		g.mu.Unlock()

		if len(rounds) == 0 {
			time.Sleep(4 * time.Second)
		} else {
			time.Sleep(10 * time.Second)
		}

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
		bonuses := make(map[string][]Bonus)
		var tookOn int
		var prima bool

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

		tookOn = toBid
		prima = !round2 || toBid == dealer

		g.mu.Lock()
		for _, p := range g.players {
			websocket.JSON.Send(p.conn, MakeMessage("trumps", TrumpsMessage{
				Trumps:     int(trumps),
				TookOn:     tookOn,
				ExtraCards: extra[p.name],
			}))
		}
		g.mu.Unlock()

		time.Sleep(time.Second)

		for k, v := range extra {
			for _, vv := range v {
				hands[k] = append(hands[k], vv)
			}
		}
		extra = nil

		wonCards := make(map[string][]Card)
		announcedBonuses := make(map[string]AnnouncedBonus)

		toPlay := (dealer + 1) % g.playerCount
		for trickNum := 0; trickNum < 8; trickNum++ {

			trick := make([]Card, 0, g.playerCount)

			for trickPlayer := 0; trickPlayer < g.playerCount; trickPlayer++ {
				trickPlayerIdx := (toPlay + trickPlayer) % g.playerCount
				g.mu.Lock()
				playerConn := g.players[trickPlayerIdx].conn
				playerName := g.players[trickPlayerIdx].name
				g.mu.Unlock()

				// If the player has a twenty or fifty, tell the client the player
				// can announce it.
				var announceBonus string
				if trumps != SuitUnknown {
					if trickNum == 0 {
						bestRun := getBestRun(hands[playerName], trumps)
						if len(bestRun) == 3 {
							announceBonus = BonusTwenty.String()
						} else if len(bestRun) == 4 {
							announceBonus = BonusFifty.String()
						}
					}
				}
				websocket.JSON.Send(playerConn, MakeMessage("your_turn", YourTurnMessage{
					AnnounceBonus: announceBonus,
				}))

				var maxTrickTrump int
				for _, t := range trick {
					if t.suit != trumps {
						continue
					}
					if t.rank.TrumpRank() > maxTrickTrump {
						maxTrickTrump = t.rank.TrumpRank()
					}
				}

				canPlay := make(map[Card]bool)
				if len(trick) == 0 {
					for _, c := range hands[playerName] {
						canPlay[c] = true
					}
				} else {
					for _, c := range hands[playerName] {
						if c.suit == trick[0].suit {
							canPlay[c] = true
						}
					}

					if len(canPlay) == 0 {
						for _, c := range hands[playerName] {
							if c.suit == trumps {
								if maxTrickTrump == 0 || c.rank.TrumpRank() > maxTrickTrump {
									canPlay[c] = true
								}
							}
						}
					}

					if len(canPlay) == 0 {
						for _, c := range hands[playerName] {
							canPlay[c] = true
						}
					}
				}

				for m := range g.ch {
					if m.Message.Type == "announce_bonus" {
						if m.Conn != playerConn {
							g.errCh <- errors.New("it's not your turn")
							continue
						}

						if trumps == SuitUnknown {
							g.errCh <- errors.New("can't announce twenty or fifty in no trumps")
							continue
						}

						bestRun := getBestRun(hands[playerName], trumps)
						var announceBonus Bonus
						if len(bestRun) == 3 {
							announceBonus = BonusTwenty
						} else if len(bestRun) == 4 {
							announceBonus = BonusFifty
						}
						if announceBonus == BonusUnknown {
							g.errCh <- errors.New("no bonus to announce")
							continue
						}
						announcedBonuses[playerName] = AnnouncedBonus{
							Bonus: announceBonus,
							Cards: bestRun,
						}

						g.errCh <- nil

						g.mu.Lock()
						for _, p := range g.players {
							websocket.JSON.Send(p.conn, MakeMessage("speech", SpeechMessage{
								Player:  toPlay,
								Message: announceBonus.String(),
							}))
						}
						g.mu.Unlock()

						continue
					}

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

					// Play the card and remove it from the player's hand.
					trick = append(trick, playMessage.Card)
					newHand := make([]Card, 0, len(hands[playerName])-1)
					for _, c := range hands[playerName] {
						if c == playMessage.Card {
							continue
						}
						newHand = append(newHand, c)
					}
					hands[playerName] = newHand

					// Send the updated trick to all players.
					g.mu.Lock()
					for _, p := range g.players {
						websocket.JSON.Send(p.conn, MakeMessage("trick", TrickMessage{
							PlayerCount: g.playerCount,
							FirstPlayer: toPlay,
							Cards:       trick,
						}))
					}
					g.mu.Unlock()

					break
				}
			}

			// Everyone's played their cards. Who wins?
			bestPlayer := 0
			bestCard := trick[bestPlayer]
			for i := 1; i < len(trick); i++ {
				c := trick[i]
				if c.suit == trumps {
					if bestCard.suit != trumps {
						bestCard = c
						bestPlayer = i
						continue
					} else if c.rank.TrumpRank() > bestCard.rank.TrumpRank() {
						bestCard = c
						bestPlayer = i
						continue
					}
					continue
				}
				if c.suit == bestCard.suit && c.rank.NonTrumpRank() > bestCard.rank.NonTrumpRank() {
					bestCard = c
					bestPlayer = i
					continue
				}
			}

			// Once the first trick is played, and before the cards are given
			// to the winning player, we need to settle twenties and fifties.
			if trumps != SuitUnknown && trickNum == 0 && len(announcedBonuses) > 0 {
				var haveFifty bool
				for _, v := range announcedBonuses {
					if v.Bonus == BonusFifty {
						haveFifty = true
					}
				}

				var highCard Card
				var highCardPlayer int
				var highCardPlayerName string

				g.mu.Lock()
				for i := 0; i < g.playerCount; i++ {
					time.Sleep(2 * time.Second)

					announcer := (dealer + 1 + i) % g.playerCount
					announcerName := g.players[announcer].name
					b := announcedBonuses[announcerName]
					if b.Bonus == BonusUnknown {
						continue
					}
					if b.Bonus == BonusTwenty && haveFifty {
						continue
					}

					if highCard.rank > b.HighCard().rank ||
						(highCard.rank == b.HighCard().rank && highCard.suit == trumps) {

						for _, p := range g.players {
							websocket.JSON.Send(p.conn, MakeMessage("speech", SpeechMessage{
								Player:  announcer,
								Message: "Yours.",
							}))
						}

						continue
					}

					highCard = b.HighCard()
					highCardPlayer = announcer
					highCardPlayerName = announcerName

					message := "My " + b.Bonus.String() + " is " + highCard.rank.String() + " high"
					if highCard.suit == trumps {
						message += " in trumps"
					}
					message += "."

					for _, p := range g.players {
						websocket.JSON.Send(p.conn, MakeMessage("speech", SpeechMessage{
							Player:  announcer,
							Message: message,
						}))
					}
				}

				time.Sleep(2 * time.Second)

				bonuses[highCardPlayerName] = append(bonuses[highCardPlayerName],
					announcedBonuses[highCardPlayerName].Bonus)

				for _, p := range g.players {
					websocket.JSON.Send(p.conn, MakeMessage("bonus_awarded", BonusAwardedMessage{
						Player:       highCardPlayer,
						Bonus:        announcedBonuses[highCardPlayerName].Bonus.String(),
						Cards:        announcedBonuses[highCardPlayerName].Cards,
						CurrentTrick: trick,
					}))
				}

				g.mu.Unlock()
			}

			time.Sleep(2 * time.Second)

			winner := (toPlay + bestPlayer) % g.playerCount

			g.mu.Lock()
			bestPlayerName := g.players[winner].name
			for _, c := range trick {
				if c.suit == trumps && c.rank == RankNine {
					bonuses[bestPlayerName] = append(bonuses[bestPlayerName], BonusMinel)
				}
				if c.suit == trumps && c.rank == RankJack {
					bonuses[bestPlayerName] = append(bonuses[bestPlayerName], BonusJass)
				}
				wonCards[bestPlayerName] = append(wonCards[bestPlayerName], c)
			}
			for _, p := range g.players {
				websocket.JSON.Send(p.conn, MakeMessage("trick_won", TrickWonMessage{
					PlayerCount: g.playerCount,
					FirstPlayer: toPlay,
					Winner:      bestPlayer,
				}))
			}
			g.mu.Unlock()

			trick = nil
			toPlay = winner
		}

		// Last trick.
		g.mu.Lock()
		bonuses[g.players[toPlay].name] = append(
			bonuses[g.players[toPlay].name], BonusStoch)
		g.mu.Unlock()

		// Calculate scores.
		roundScores := make(map[string]int)
		for k, v := range wonCards {
			for _, vv := range v {
				switch vv.rank {
				case RankJack:
					roundScores[k] += 2
				case RankQueen:
					roundScores[k] += 3
				case RankKing:
					roundScores[k] += 4
				case RankTen:
					roundScores[k] += 10
				case RankAce:
					roundScores[k] += 11
				}
			}
		}
		for k, v := range bonuses {
			for _, vv := range v {
				roundScores[k] += vv.Value()
			}
		}

		var roundWinner string
		var winningScore int
		for k, v := range roundScores {
			if v > winningScore {
				roundWinner = k
				winningScore = v
			}
		}

		var roundWinnerIdx int
		g.mu.Lock()
		for i, p := range g.players {
			if p.name == roundWinner {
				roundWinnerIdx = i
				break
			}
		}
		g.mu.Unlock()

		var tookOnName string
		g.mu.Lock()
		tookOnName = g.players[tookOn].name
		g.mu.Unlock()

		if g.playerCount == 2 {
			if roundWinnerIdx != tookOn {
				roundScores[roundWinner] += roundScores[tookOnName]
				roundScores[tookOnName] = 0
			}
		} else if g.playerCount == 3 {
			// TODO: bonus rounds

			g.mu.Lock()
			for _, p := range g.players {
				if roundWinnerIdx == tookOn {
					if p.name == roundWinner {
						roundScores[p.name] = 2
					} else {
						roundScores[p.name] = -1
					}
				} else {
					mod := 1
					if prima {
						mod = 2
					}
					if p.name == tookOnName {
						roundScores[p.name] = -2 * mod
					} else {
						roundScores[p.name] = 1 * mod
					}
				}
			}
			g.mu.Unlock()
		} else {
			// TODO: 4 players
		}

		round := make([]int, 0, g.playerCount)
		g.mu.Lock()
		for _, p := range g.players {
			round = append(round, roundScores[p.name])
			scores[p.name] += roundScores[p.name]
		}
		g.mu.Unlock()
		rounds = append(rounds, round)

		dealer = (dealer + 1) % g.playerCount
	}
}

func getBestRun(hand []Card, trumps Suit) []Card {
	sort.Slice(hand, func(i, j int) bool {
		if hand[i].suit == hand[j].suit {
			return hand[i].rank < hand[j].rank
		}
		return hand[i].suit < hand[j].suit
	})

	var bestRun []Card
	var highCard Card
	var run []Card
	for i, c := range hand {
		var appended bool
		if len(run) == 0 ||
			(c.suit == run[0].suit && c.rank == run[len(run)-1].rank+1) {
			appended = true
			run = append(run, c)
		}

		if i == len(hand)-1 || !appended {
			if len(run) == 3 {
				if bestRun == nil ||
					(len(bestRun) == 3 && run[len(run)-1].rank > highCard.rank) ||
					(len(bestRun) == 3 && run[len(run)-1].rank == highCard.rank && run[0].suit == trumps) {
					bestRun = make([]Card, 3)
					for i, cc := range run {
						bestRun[i] = cc
						highCard = cc
					}
				}
			} else if len(run) > 3 {
				if bestRun == nil ||
					run[len(run)-1].rank > highCard.rank ||
					(run[len(run)-1].rank == highCard.rank && run[0].suit == trumps) {
					bestRun = make([]Card, 4)
					for i, cc := range run[len(run)-4:] {
						bestRun[i] = cc
						highCard = cc
					}
				}
			}

			run = nil
			if !appended {
				run = append(run, c)
			}
		}
	}

	return bestRun
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
