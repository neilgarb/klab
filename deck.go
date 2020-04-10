package klab

import (
	"math/rand"
	"sync"
)

type Deck struct {
	size int

	mu sync.Mutex
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
	return &Deck{size: len(cards), cards: cards}
}

func (d *Deck) Size() int {
	return d.size
}

func (d *Deck) Shuffle() {
	d.mu.Lock()
	defer d.mu.Unlock()

	rand.Shuffle(len(d.cards), func(i, j int) {
		d.cards[i], d.cards[j] = d.cards[j], d.cards[i]
	})
}

func (d *Deck) Deal(count int) []Card {
	d.mu.Lock()
	defer d.mu.Unlock()

	cards := d.cards[:count]
	d.cards = d.cards[count:]
	return cards
}