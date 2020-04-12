package klab

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestGetBestRun(t *testing.T) {
	testCases := []struct {
		hand    []Card
		trumps  Suit
		bestRun []Card
	}{
		{
			hand: []Card{
				{SuitDiamonds, RankEight},
				{SuitDiamonds, RankNine},
				{SuitHearts, RankTen},
			},
		},
		{
			hand: []Card{
				{SuitDiamonds, RankEight},
				{SuitDiamonds, RankNine},
				{SuitDiamonds, RankTen},
			},
			bestRun: []Card{
				{SuitDiamonds, RankEight},
				{SuitDiamonds, RankNine},
				{SuitDiamonds, RankTen},
			},
		},
		{
			hand: []Card{
				{SuitDiamonds, RankEight},
				{SuitDiamonds, RankNine},
				{SuitDiamonds, RankTen},
			},
			bestRun: []Card{
				{SuitDiamonds, RankEight},
				{SuitDiamonds, RankNine},
				{SuitDiamonds, RankTen},
			},
		},
		{
			hand: []Card{
				{SuitDiamonds, RankTen},
				{SuitDiamonds, RankEight},
				{SuitHearts, RankJack},
				{SuitDiamonds, RankNine},
			},
			bestRun: []Card{
				{SuitDiamonds, RankEight},
				{SuitDiamonds, RankNine},
				{SuitDiamonds, RankTen},
			},
		},
		{
			hand: []Card{
				{SuitDiamonds, RankTen},
				{SuitDiamonds, RankEight},
				{SuitDiamonds, RankJack},
				{SuitDiamonds, RankNine},
			},
			bestRun: []Card{
				{SuitDiamonds, RankEight},
				{SuitDiamonds, RankNine},
				{SuitDiamonds, RankTen},
				{SuitDiamonds, RankJack},
			},
		},
		{
			hand: []Card{
				{SuitDiamonds, RankEight},
				{SuitDiamonds, RankNine},
				{SuitDiamonds, RankTen},
				{SuitDiamonds, RankJack},
				{SuitDiamonds, RankQueen},
			},
			bestRun: []Card{
				{SuitDiamonds, RankNine},
				{SuitDiamonds, RankTen},
				{SuitDiamonds, RankJack},
				{SuitDiamonds, RankQueen},
			},
		},
		{
			hand: []Card{
				{SuitDiamonds, RankEight},
				{SuitDiamonds, RankNine},
				{SuitDiamonds, RankTen},
				{SuitSpades, RankEight},
				{SuitSpades, RankNine},
				{SuitSpades, RankTen},
			},
			trumps: SuitDiamonds,
			bestRun: []Card{
				{SuitDiamonds, RankEight},
				{SuitDiamonds, RankNine},
				{SuitDiamonds, RankTen},
			},
		},
		{
			hand: []Card{
				{SuitDiamonds, RankEight},
				{SuitDiamonds, RankNine},
				{SuitDiamonds, RankTen},
				{SuitSpades, RankEight},
				{SuitSpades, RankNine},
				{SuitSpades, RankTen},
			},
			trumps: SuitSpades,
			bestRun: []Card{
				{SuitSpades, RankEight},
				{SuitSpades, RankNine},
				{SuitSpades, RankTen},
			},
		},
		{
			hand: []Card{
				{SuitDiamonds, RankEight},
				{SuitDiamonds, RankNine},
				{SuitDiamonds, RankTen},
				{SuitSpades, RankEight},
				{SuitSpades, RankNine},
				{SuitSpades, RankTen},
			},
			bestRun: []Card{
				{SuitDiamonds, RankEight},
				{SuitDiamonds, RankNine},
				{SuitDiamonds, RankTen},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run("", func(t *testing.T) {
			bestRun := getBestRun(testCase.hand, testCase.trumps)
			assert.Equal(t, testCase.bestRun, bestRun)
		})
	}
}
