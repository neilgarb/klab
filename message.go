package klab

import "encoding/json"

type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type ErrorMessage string

type CreateGameMessage struct {
	Name        string `json:"name"`
	PlayerCount int    `json:"player_count"`
}

type JoinGameMessage struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type GameLobbyMessage struct {
	Code        string   `json:"code"`
	Host        bool     `json:"host"`
	CanStart    bool     `json:"can_start"`
	PlayerCount int      `json:"player_count"`
	PlayerNames []string `json:"player_names"`
}

type GameStartedMessage struct {
	Name        string   `json:"name"`
	PlayerNames []string `json:"player_names"`
}

type GameScoresMessage struct {
	PlayerNames []string `json:"player_names"`
	Total       []int    `json:"total"`
	Scores      [][]int  `json:"scores"`
}

type RoundStartedMessage struct {
	Dealer int `json:"dealer"`
}

type RoundDealtMessage struct {
	PlayerCount int    `json:"player_count"`
	Dealer      int    `json:"dealer"`
	DeckSize    int    `json:"deck_size"`
	Cards       []Card `json:"cards"`
	CardUp      Card   `json:"card_up"`
}

type BidRequestMessage struct {
	CardUp Card `json:"card_up"`
	Round2 bool `json:"round2"`
	Bimah  bool `json:"bimah"`
}

type SpeechMessage struct {
	Player  int    `json:"player"`
	Message string `json:"message"`
}

type BidMessage struct {
	Pass bool `json:"pass"`
	Suit int  `json:"suit"`
}

type TrumpsMessage struct {
	Trumps     int    `json:"trumps"`
	ExtraCards []Card `json:"extra_cards"`
}

type YourTurnMessage struct {
}

type PlayMessage struct {
	Card Card `json:"card"`
}

type TrickMessage struct {
	PlayerCount int    `json:"player_count"`
	FirstPlayer int    `json:"first_player"`
	Cards       []Card `json:"cards"`
}

type TrickWonMessage struct {
	PlayerCount int `json:"player_count"`
	FirstPlayer int `json:"first_player"`
	Winner      int `json:"winner"`
}

func MakeMessage(typ string, data interface{}) *Message {
	b, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	return &Message{typ, b}
}
