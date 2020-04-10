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
	suit Suit `json:"suit"`
	rank Rank `json:"rank"`
}

func NewCard(suit Suit, rank Rank) Card {
	return Card{suit, rank}
}

func (c Card) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{
		Suit int `json:"suit"`
		Rank int `json:"rank"`
	}{int(c.suit), int(c.rank)})
}