package klab

type Bonus int

const (
	BonusUnknown Bonus = 0
	BonusTwenty  Bonus = 1
	BonusFifty   Bonus = 2
	BonusBella   Bonus = 3
	BonusStoch   Bonus = 4
	BonusMinel   Bonus = 5
	BonusJass    Bonus = 6
)

func (b Bonus) String() string {
	switch b {
	case BonusTwenty:
		return "Twenty"
	case BonusFifty:
		return "Fifty"
	case BonusBella:
		return "Bella"
	case BonusStoch:
		return "Stoch"
	case BonusMinel:
		return "Minel"
	case BonusJass:
		return "Jass"
	}
	return "Unknown"
}

func (b Bonus) Value() int {
	switch b {
	case BonusTwenty:
		return 20
	case BonusFifty:
		return 50
	case BonusBella:
		return 20
	case BonusStoch:
		return 10
	case BonusMinel:
		return 14
	case BonusJass:
		return 20
	}
	return 0
}

type AnnouncedBonus struct {
	Bonus Bonus
	Cards []Card
}

func (a AnnouncedBonus) HighCard() Card {
	if len(a.Cards) == 0 {
		return Card{}
	}
	return a.Cards[len(a.Cards)-1]
}
