package klab

import "math/rand"

type Deck struct {
	cards []Card
}

func NewDeck(withSevens bool) *Deck {
	cards := make([]Card, 0, 32)
	for _, s := range AllSuits() {
		for _, r := range AllRanks() {
			if r == RankSeven && !withSevens {
				continue
			}
			cards = append(cards, NewCard(s, r))
		}
	}
	return &Deck{cards}
}

func (d *Deck) Shuffle() {
	rand.Shuffle(len(d.cards), func(i, j int) {
		d.cards[i], d.cards[j] = d.cards[j], d.cards[i]
	})
}