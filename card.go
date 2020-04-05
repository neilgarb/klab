package klab

type Suit int

const (
	SuitUnknown  Suit = 0
	SuitHearts   Suit = 1
	SuitDiamonds Suit = 2
	SuitClubs    Suit = 3
	SuitSpades   Suit = 4
)

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
	suit Suit
	rank Rank
}

func NewCard(suit Suit, rank Rank) Card {
	return Card{suit,rank}
}
