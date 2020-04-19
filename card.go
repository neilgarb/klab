package klab

import "encoding/json"

type Suit int

const (
	SuitUnknown  Suit = 0
	SuitClubs    Suit = 1
	SuitHearts   Suit = 2
	SuitSpades   Suit = 3
	SuitDiamonds Suit = 4
)

func (s Suit) String() string {
	switch s {
	case SuitUnknown:
		return "No trumps"
	case SuitClubs:
		return "Clubs"
	case SuitHearts:
		return "Hearts"
	case SuitSpades:
		return "Spades"
	case SuitDiamonds:
		return "Diamonds"
	}
	return "Unknown"
}

func AllSuits() []Suit {
	return []Suit{
		SuitClubs,
		SuitHearts,
		SuitSpades,
		SuitDiamonds,
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

func (r Rank) NonTrumpRank() int {
	switch r {
	case RankSeven:
		return 1
	case RankEight:
		return 2
	case RankNine:
		return 3
	case RankJack:
		return 4
	case RankQueen:
		return 5
	case RankKing:
		return 6
	case RankTen:
		return 7
	case RankAce:
		return 8
	}
	return 0
}

func (r Rank) TrumpRank() int {
	switch r {
	case RankSeven:
		return 1
	case RankEight:
		return 2
	case RankQueen:
		return 3
	case RankKing:
		return 4
	case RankTen:
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

func (c Card) String() string {
	return c.rank.String() + c.suit.String()
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
