package klab

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	roundCount  int
	maxScore    int

	mu      sync.Mutex
	host    string
	players []*Player
	started bool
	ch      chan *ConnMessage
	errCh   chan error
}

func NewGame(code string) (*Game, error) {
	code = strings.TrimSpace(strings.ToUpper(code))
	if code == "" {
		return nil, errors.New("no game code provided")
	}

	//if playerCount < 2 || playerCount > 4 {
	//	return nil, errors.New("number of players should be 2, 3 or 4")
	//}
	//
	//if playerCount == 2 || playerCount == 4 {
	//	if maxScore != 101 && maxScore != 501 && maxScore != 1001 && maxScore != 1501 {
	//		return nil, errors.New("invalid max score")
	//	}
	//} else {
	//	if roundCount != 9 && roundCount != 12 && roundCount != 15 && roundCount != 18 {
	//		return nil, errors.New("invalid round count")
	//	}
	//}

	return &Game{
		code: code,
		//playerCount: playerCount,
		//roundCount:  roundCount,
		//maxScore:    maxScore,
		roundCount: 15,
		maxScore:   1001,
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

	if len(g.players) == 4 {
		return errors.New("this game is full")
	}

	player, err := NewPlayer(conn, name)
	if err != nil {
		return err
	}
	g.players = append(g.players, player)
	playerNames = append(playerNames, player.name)

	if len(g.players) == 1 {
		g.host = player.name
	}
	/*
		var gameDescription string
		if g.playerCount == 2 || g.playerCount == 4 {
			gameDescription = fmt.Sprintf("Play until one player has at least %d points", g.maxScore)
		} else if g.playerCount == 3 {
			gameDescription = fmt.Sprintf("Play %d single rounds and 3 double rounds", g.roundCount-3)
		}
	*/

	for _, p := range g.players {
		g.send(p.conn, "game_lobby", GameLobbyMessage{
			Code: g.code,
			Host: p.name == g.host,
			//CanStart:        g.playerCount == len(g.players),
			PlayerCount: len(g.players),
			PlayerNames: playerNames,
			//GameDescription: gameDescription,
			Name: p.name,
		})
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

	for _, p := range g.players {
		g.send(p.conn, "game_lobby", GameLobbyMessage{
			Code:        g.code,
			Host:        p.name == g.host,
			CanStart:    g.playerCount == len(g.players),
			PlayerCount: g.playerCount,
			PlayerNames: playerNames,
		})
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

	g.playerCount = len(g.players)
	g.started = true

	var playerNames []string
	for _, p := range g.players {
		playerNames = append(playerNames, p.name)
	}

	for _, p := range g.players {
		g.send(p.conn, "game_started", GameStartedMessage{
			Name:        p.name,
			PlayerNames: playerNames,
		})
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

	scores := make(map[int]int)
	var rounds [][]int

	dealer := rand.Intn(g.playerCount)

	for {
		if len(rounds) == 0 {
			time.Sleep(4 * time.Second)
		} else {
			time.Sleep(10 * time.Second)
		}

		g.mu.Lock()
		for _, p := range g.players {
			g.send(p.conn, "round_started", RoundStartedMessage{
				Dealer: dealer,
			})
		}
		g.mu.Unlock()

		time.Sleep(time.Second)

		deck := NewDeck(g.playerCount != 3)
		deck.Shuffle()

		hands := make(map[string][]Card)
		extra := make(map[string][]Card)
		var tookOn int
		var prima bool

		g.mu.Lock()
		for _, p := range g.players {
			hands[p.name] = append(hands[p.name], deck.Deal(3)...)
		}
		for _, p := range g.players {
			hands[p.name] = append(hands[p.name], deck.Deal(3)...)
		}

		var cardUp Card
		var extraCount int
		if g.playerCount == 4 {
			cardUp = Card{suit: AllSuits()[dealer], rank: RankSeven}
			extraCount = 2
		} else {
			cardUp = deck.Deal(1)[0]
			extraCount = 3
		}
		for _, p := range g.players {
			extra[p.name] = append(extra[p.name], deck.Deal(extraCount)...)
		}

		for _, p := range g.players {
			g.send(p.conn, "round_dealt", RoundDealtMessage{
				PlayerCount: len(g.players),
				Dealer:      dealer,
				DeckSize:    deck.Size(),
				Cards:       hands[p.name],
				CardUp:      cardUp,
				Suits:       AllSuits(),
			})
		}
		g.mu.Unlock()

		time.Sleep(5 * time.Second)

		var trumps Suit
		var pool bool
		var round2 bool
		toBid := (dealer + 1) % g.playerCount

	BidLoop:
		for {
			g.mu.Lock()
			bidderConn := g.players[toBid].conn
			g.mu.Unlock()

			g.send(bidderConn, "bid_request", BidRequestMessage{
				CardUp:  cardUp,
				Round2:  round2,
				Bimah:   round2 && toBid == dealer,
				CanPool: g.playerCount == 3,
			})

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
				if bidMessage.Pool && g.playerCount != 3 {
					g.errCh <- errors.New("you can only pool in a 3-player game")
					continue
				}

				if !round2 {
					if bidMessage.Pass {
					} else if Suit(bidMessage.Suit) != cardUp.suit {
						g.errCh <- errors.New("must pass or take on")
						continue
					} else {
						trumps = cardUp.suit
						pool = bidMessage.Pool
						g.errCh <- nil

						trumpsMessage := randomPlayMessage()
						if pool {
							trumpsMessage = "I pool you."
						}
						g.mu.Lock()
						for _, p := range g.players {
							g.send(p.conn, "speech", SpeechMessage{
								Player:  toBid,
								Message: trumpsMessage,
							})
						}
						g.mu.Unlock()

						break BidLoop
					}
					if toBid == dealer {
						round2 = true
					}
					g.errCh <- nil

					passMessage := randomPassMessage()
					g.mu.Lock()
					for _, p := range g.players {
						g.send(p.conn, "speech", SpeechMessage{
							Player:  toBid,
							Message: passMessage,
						})
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
						pool = bidMessage.Pool
						g.errCh <- nil

						trumpsMessage := trumps.String()
						if pool {
							trumpsMessage = fmt.Sprintf("I pool you in %s.", trumps.String())
						}
						g.mu.Lock()
						for _, p := range g.players {
							g.send(p.conn, "speech", SpeechMessage{
								Player:  toBid,
								Message: trumpsMessage,
							})
						}
						g.mu.Unlock()

						break BidLoop
					}

					g.errCh <- nil

					passMessage := randomPassMessage()
					g.mu.Lock()
					for _, p := range g.players {
						g.send(p.conn, "speech", SpeechMessage{
							Player:  toBid,
							Message: passMessage,
						})
					}
					g.mu.Unlock()

					toBid = (toBid + 1) % g.playerCount
					continue BidLoop
				}

				trumps = Suit(bidMessage.Suit)
				pool = bidMessage.Pool
				g.errCh <- nil

				trumpsMessage := trumps.String()
				if pool {
					trumpsMessage = fmt.Sprintf("I pool you in %s.", trumps.String())
				}
				g.mu.Lock()
				for _, p := range g.players {
					g.send(p.conn, "speech", SpeechMessage{
						Player:  toBid,
						Message: trumpsMessage,
					})
				}
				g.mu.Unlock()

				break BidLoop
			}
		}

		tookOn = toBid
		prima = g.playerCount == 3 && (!round2 || toBid != dealer)

		g.mu.Lock()
		for _, p := range g.players {
			g.send(p.conn, "trumps", TrumpsMessage{
				Trumps:     int(trumps),
				TookOn:     tookOn,
				Prima:      prima,
				Pooled:     pool,
				ExtraCards: extra[p.name],
			})
		}
		g.mu.Unlock()

		time.Sleep(time.Second)

		for k, v := range extra {
			for _, vv := range v {
				hands[k] = append(hands[k], vv)
			}
		}
		extra = nil

		wonCards := make(map[int][]Card)
		bonuses := make(map[int][]Bonus)
		announcedBonuses := make(map[string]AnnouncedBonus)
		bellaPlayed := make(map[Rank]string)

		toPlay := (dealer + 1) % g.playerCount
		trickCount := 9
		if g.playerCount == 4 {
			trickCount = 8
		}
		for trickNum := 0; trickNum < trickCount; trickNum++ {

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
				g.send(playerConn, "your_turn", YourTurnMessage{
					AnnounceBonus: announceBonus,
				})

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
							g.send(p.conn, "speech", SpeechMessage{
								Player:  trickPlayerIdx,
								Message: announceBonus.String(),
							})
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

					// Bella.
					if playMessage.Card.suit == trumps {
						var otherRank Rank
						if playMessage.Card.rank == RankQueen {
							otherRank = RankKing
						} else if playMessage.Card.rank == RankKing {
							otherRank = RankQueen
						}
						if otherRank > 0 {
							if bellaPlayed[otherRank] == playerName {
								bonuses[trickPlayerIdx] = append(
									bonuses[trickPlayerIdx], BonusBella)
								g.mu.Lock()
								for _, p := range g.players {
									g.send(p.conn, "speech", SpeechMessage{
										Player:  trickPlayerIdx,
										Message: "Bella",
									})
								}
								g.mu.Unlock()
							}
							bellaPlayed[playMessage.Card.rank] = playerName
						}
					}

					// Send the updated trick to all players.
					g.mu.Lock()
					for _, p := range g.players {
						g.send(p.conn, "trick", TrickMessage{
							PlayerCount: g.playerCount,
							FirstPlayer: toPlay,
							Cards:       trick,
						})
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
							g.send(p.conn, "speech", SpeechMessage{
								Player:  announcer,
								Message: "It's yours.",
							})
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
						g.send(p.conn, "speech", SpeechMessage{
							Player:  announcer,
							Message: message,
						})
					}
				}

				time.Sleep(2 * time.Second)

				bonuses[highCardPlayer] = append(bonuses[highCardPlayer],
					announcedBonuses[highCardPlayerName].Bonus)

				for _, p := range g.players {
					g.send(p.conn, "bonus_awarded", BonusAwardedMessage{
						Player:       highCardPlayer,
						Bonus:        announcedBonuses[highCardPlayerName].Bonus.String(),
						Cards:        announcedBonuses[highCardPlayerName].Cards,
						CurrentTrick: trick,
					})
				}

				g.mu.Unlock()
			}

			time.Sleep(2 * time.Second)

			winner := (toPlay + bestPlayer) % g.playerCount

			for _, c := range trick {
				if c.suit == trumps && c.rank == RankNine {
					bonuses[winner] = append(bonuses[winner], BonusMinel)
				}
				if c.suit == trumps && c.rank == RankJack {
					bonuses[winner] = append(bonuses[winner], BonusJass)
				}
				wonCards[winner] = append(wonCards[winner], c)
			}

			g.mu.Lock()
			for _, p := range g.players {
				g.send(p.conn, "trick_won", TrickWonMessage{
					PlayerCount: g.playerCount,
					FirstPlayer: toPlay,
					Winner:      bestPlayer,
				})
			}
			g.mu.Unlock()

			trick = nil
			toPlay = winner
		}

		time.Sleep(2 * time.Second)

		// Last trick.
		bonuses[toPlay] = append(bonuses[toPlay], BonusStoch)

		// Calculate teams for the round.
		var teams [][]int
		if g.playerCount == 2 {
			teams = [][]int{{0}, {1}}
		} else if g.playerCount == 3 {
			if pool {
				teams = [][]int{{tookOn}, {(tookOn + 1) % 3, (tookOn + 2) % 3}}
			} else {
				teams = [][]int{{0}, {1}, {2}}
			}
		} else {
			teams = [][]int{{0, 2}, {1, 3}}
		}
		teamMap := make(map[int]int) // player -> team
		for i, t := range teams {
			for _, tt := range t {
				teamMap[tt] = i
			}
		}

		// Assign won cards and bonuses to teams.
		roundWonCards := make(map[int][]Card) // team -> cards
		roundBonuses := make(map[int][]Bonus) // team -> bonuses
		for k, v := range wonCards {
			roundWonCards[teamMap[k]] = append(roundWonCards[teamMap[k]], v...)
		}
		for k, v := range bonuses {
			roundBonuses[teamMap[k]] = append(roundBonuses[teamMap[k]], v...)
		}

		// Calculate team scores.
		roundScoresMessage := RoundScoresMessage{
			Title:  fmt.Sprintf("Round %d scores", len(rounds)+1),
			Scores: make(map[int]RoundScores),
			TookOn: teamMap[tookOn],
			Pooled: pool,
			Prima:  prima,
		}
		roundScores := make(map[int]int)
		for k, v := range roundWonCards {
			for _, vv := range v {
				var score int
				switch vv.rank {
				case RankJack:
					score = 2
				case RankQueen:
					score = 3
				case RankKing:
					score = 4
				case RankTen:
					score = 10
				case RankAce:
					score = 11
				}
				roundScores[k] += score
				m := roundScoresMessage.Scores[k]
				m.WonCards = append(m.WonCards, RoundScoreCard{score, vv})
				roundScoresMessage.Scores[k] = m
			}
		}
		for k, v := range roundBonuses {
			for _, vv := range v {
				roundScores[k] += vv.Value()
				m := roundScoresMessage.Scores[k]
				m.Bonuses = append(m.Bonuses, RoundScoreBonus{vv.Value(), vv.String()})
				roundScoresMessage.Scores[k] = m
			}
		}

		// Send round scores.
		g.mu.Lock()
		var teamNames []string
		for _, t := range teams {
			var names []string
			for _, tt := range t {
				names = append(names, g.players[tt].name)
			}
			teamNames = append(teamNames, strings.Join(names, " & "))
		}
		roundScoresMessage.PlayerNames = teamNames
		for _, p := range g.players {
			g.send(p.conn, "round_scores", roundScoresMessage)
		}
		g.mu.Unlock()

		time.Sleep(15 * time.Second)

		// Which team won?
		winningTeam := -1
		winningScore := -1
		for k, v := range roundScores {
			if v > winningScore {
				winningTeam = k
				winningScore = v
			} else if v == winningScore && teamMap[tookOn] != k {
				winningTeam = k
			}
		}

		// Assign game scores.
		gameScores := make(map[int]int) // player -> score
		var round []int
		g.mu.Lock()
		if g.playerCount == 3 {
			mod := 1
			if pool {
				mod *= 2
			}
			if g.roundCount-len(rounds) <= 3 {
				mod *= 2
			}
			if prima && teamMap[tookOn] != winningTeam {
				mod *= 2
			}
			if teamMap[tookOn] == winningTeam {
				gameScores[tookOn] = 2 * mod
				gameScores[(tookOn+1)%3] = -1 * mod
				gameScores[(tookOn+2)%3] = -1 * mod
			} else {
				gameScores[tookOn] = -2 * mod
				gameScores[(tookOn+1)%3] = 1 * mod
				gameScores[(tookOn+2)%3] = 1 * mod
			}
			for i := range g.players {
				round = append(round, gameScores[i])
			}
		} else {
			if teamMap[tookOn] != winningTeam {
				roundScores[(teamMap[tookOn]+1)%2] += roundScores[teamMap[tookOn]]
				roundScores[teamMap[tookOn]] = 0
			}
			for t := range teams {
				round = append(round, roundScores[t])
			}
		}
		g.mu.Unlock()

		rounds = append(rounds, round)
		for k, v := range round {
			scores[k] += v
		}

		// Send game scores.
		g.mu.Lock()
		var total []int
		if g.playerCount == 4 {
			playerNames = []string{
				g.players[0].name + " & " + g.players[2].name,
				g.players[1].name + " & " + g.players[3].name,
			}
			total = []int{scores[0], scores[1]}
		} else {
			var playerNames []string
			for i, p := range g.players {
				playerNames = append(playerNames, p.name)
				total = append(total, scores[i])
			}
		}
		for _, p := range g.players {
			g.send(p.conn, "game_scores", GameScoresMessage{
				PlayerNames: playerNames,
				Scores:      rounds,
				Total:       total,
			})
		}
		g.mu.Unlock()

		maxScore := -1
		var haveTie bool
		for _, v := range scores {
			if v > maxScore {
				maxScore = v
			} else if v == maxScore {
				haveTie = true
			}
		}
		if g.playerCount == 2 || g.playerCount == 4 {
			if maxScore >= g.maxScore {
				break
			}
		} else if len(rounds) == g.roundCount && !haveTie {
			break
		}

		dealer = (dealer + 1) % g.playerCount
	}

	time.Sleep(10 * time.Second)

	g.mu.Lock()
	var gameOverMessage GameOverMessage
	if g.playerCount == 4 {
		gameOverMessage.Positions = append(gameOverMessage.Positions,
			Position{
				PlayerName: g.players[0].name + " & " + g.players[2].name,
				Score:      scores[0],
			})
		gameOverMessage.Positions = append(gameOverMessage.Positions,
			Position{
				PlayerName: g.players[1].name + " & " + g.players[3].name,
				Score:      scores[1],
			})
	} else {
		for i, p := range g.players {
			gameOverMessage.Positions = append(gameOverMessage.Positions,
				Position{
					PlayerName: p.name,
					Score:      scores[i],
				})
		}
	}
	sort.Slice(gameOverMessage.Positions, func(i, j int) bool {
		return gameOverMessage.Positions[i].Score > gameOverMessage.Positions[j].Score
	})
	for _, p := range g.players {
		g.send(p.conn, "game_over", gameOverMessage)
	}
	g.mu.Unlock()
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

func (g *Game) MaybeSay(conn *websocket.Conn, message string) (bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	player := -1
	for i, p := range g.players {
		if p.conn == conn {
			player = i
			break
		}
	}
	if player == -1 {
		return false, nil
	}

	for _, p := range g.players {
		g.send(p.conn, "speech", SpeechMessage{
			Player:  player,
			Message: message,
		})
	}

	return true, nil
}

func (g *Game) MaybeSwap(conn *websocket.Conn, newPosition int) (bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.players) < 3 {
		return false, errors.New("can't swap with this many players in the game")
	}

	player := -1
	for i, p := range g.players {
		if p.conn == conn {
			player = i
			break
		}
	}
	if player == -1 {
		return false, nil
	}

	// Pos player is swapping with newPosition
	if newPosition >= len(g.players) {
		return false, errors.New("invalid new position")
	}

	g.players[player], g.players[newPosition] =
		g.players[newPosition], g.players[player]

	var playerNames []string
	for _, p := range g.players {
		playerNames = append(playerNames, p.name)
	}

	for _, p := range g.players {
		g.send(p.conn, "game_lobby", GameLobbyMessage{
			Code:        g.code,
			Host:        p.name == g.host,
			PlayerCount: len(g.players),
			PlayerNames: playerNames,
			Name:        p.name,
		})
	}

	return true, nil
}

func (g *Game) send(conn *websocket.Conn, typ string, data interface{}) {
	log.Printf("%s -> %s: %s %+v", g.code, conn.Request().RemoteAddr, typ, data)
	websocket.JSON.Send(conn, MakeMessage(typ, data))
}

func randomPlayMessage() string {
	messages := []string{
		"Play",
		"Play",
		"Play",
		"Play",
		"Giddyup",
		"Let's go",
		"Ya why not",
		"I'll play",
	}
	return messages[rand.Intn(len(messages))]
}

func randomPassMessage() string {
	messages := []string{
		"Pass",
		"Pass",
		"Pass",
		"Pass",
		"No",
		"No thanks",
		"No no",
	}
	return messages[rand.Intn(len(messages))]
}
