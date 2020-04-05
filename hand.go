package klab

import "errors"

type RoundState int

const (
	RoundStateUnknown   RoundState = 0
	RoundStateStarted   RoundState = 1
	RoundStateBidRound1 RoundState = 2
	RoundStateBidRound2 RoundState = 3
	RoundStatePlaying   RoundState = 4
	RoundStateScore     RoundState = 5
	RoundStateComplete  RoundState = 6
)

type Round struct {
	players []*Player
	dealer  PlayerID
	state   RoundState
	trumps  Suit
}

func NewRound(players []*Player, dealer PlayerID) *Round {
	return &Round{players: players, dealer: dealer}
}

func (r *Round) Start(players []*Player, dealer PlayerID) error {
	if r.state != RoundStateUnknown {
		return errors.New("invalid state transition")
	}

	r.state = RoundStateStarted
	return nil
}