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
	Scores      [][]int  `json:"scores"`
}

type RoundStartedMessage struct {
	Dealer int `json:"dealer"`
}

func MakeMessage(typ string, data interface{}) *Message {
	b, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	return &Message{typ, b}
}
