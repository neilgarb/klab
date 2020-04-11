package klab

import "encoding/json"

type Suit int

const (
	SuitUnknown  Suit = 0
	SuitClubs    Suit = 1
	SuitDiamonds Suit = 2
	SuitHearts   Suit = 3
	SuitSpades   Suit = 4
)

func (s Suit) String() string {
	switch s {
	case SuitUnknown:
		return "No trumps"
	case SuitClubs:
		return "Clubs"
	case SuitDiamonds:
		return "Diamonds"
	case SuitHearts:
		return "Hearts"
	case SuitSpades:
		return "Spades"
	}
	return "Unknown"
}

func AllSuits() []Suit {
	return []Suit{
		SuitHearts,
		SuitDiamonds,
		SuitClubs,
		SuitSpades,
	}
}

type Rank int

const (
	RankUnknown Rank = 0
	RankSeven   Rank = 1
	RankEight   Rank = 2
	RankNine    Rank = 3
	RankTen     Rank = 4
	RankJack    Rank = 5
	RankQueen   Rank = 6
	RankKing    Rank = 7
	RankAce     Rank = 8
)

func (r Rank) String() string {
	switch r {
	case RankSeven:
		return "7"
	case RankEight:
		return "8"
	case RankNine:
		return "9"
	case RankTen:
		return "10"
	case RankJack:
		return "J"
	case RankQueen:
		return "Q"
	case RankKing:
		return "K"
	case RankAce:
		return "A"
	}
	return "Unknown"
}

func (r Rank) TrumpRank() int {
	switch r {
	case RankSeven:
		return 1
	case RankEight:
		return 2
	case RankTen:
		return 3
	case RankQueen:
		return 4
	case RankKing:
		return 5
	case RankAce:
		return 6
	case RankNine:
		return 7
	case RankJack:
		return 8
	}
	return 0
}

func AllRanks() []Rank {
	return []Rank{
		RankSeven,
		RankEight,
		RankNine,
		RankTen,
		RankJack,
		RankQueen,
		RankKing,
		RankAce,
	}
}

type Card struct {
	suit Suit
	rank Rank
}

func NewCard(suit Suit, rank Rank) Card {
	return Card{suit, rank}
}

type suitDTO struct {
	Suit int `json:"suit"`
	Rank int `json:"rank"`
}

func (c Card) MarshalJSON() ([]byte, error) {
	return json.Marshal(suitDTO{int(c.suit), int(c.rank)})
}

func (c *Card) UnmarshalJSON(b []byte) error {
	var d suitDTO
	if err := json.Unmarshal(b, &d); err != nil {
		return err
	}
	*c = Card{Suit(d.Suit), Rank(d.Rank)}
	return nil
}
