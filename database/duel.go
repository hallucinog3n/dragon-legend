package database

import "github.com/syntaxgame/dragon-legend/utils"

type Duel struct {
	EnemyID    int
	Coordinate utils.Location
	Started    bool
}
