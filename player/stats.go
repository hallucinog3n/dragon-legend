package player

import (
	"github.com/syntaxgame/dragon-legend/database"
	"github.com/syntaxgame/dragon-legend/utils"
)

type GetStatsHandler struct {
}

type AddStatHandler struct {
	amount   uint16
	statType byte
}

var (
	STAT_ADDED = utils.Packet{0xAA, 0x55, 0x0B, 0x00, 0x75, 0x00, 0x01, 0x55, 0xAA}
)

func (gsh *GetStatsHandler) Handle(s *database.Socket) ([]byte, error) {
	return s.Character.GetStats()
}

func (h *AddStatHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	h.statType = data[5]
	h.amount = uint16(utils.BytesToInt(data[6:8], true))
	return h.addStat(s)
}

func (h *AddStatHandler) addStat(s *database.Socket) ([]byte, error) {

	if s.Character != nil {
		stat := s.Stats
		if stat.StatPoints < int(h.amount) {
			return nil, nil
		}

		switch h.statType {
		case 0:
			stat.STR += int(h.amount)
			break
		case 1:
			stat.DEX += int(h.amount)
			break
		case 2:
			stat.INT += int(h.amount)
			break
		}

		stat.StatPoints -= int(h.amount)

		resp := STAT_ADDED

		location := database.ConvertPointToLocation(s.Character.Coordinate)
		resp.Insert(utils.FloatToBytes(location.X, 4, true), 7)  // location-x
		resp.Insert(utils.FloatToBytes(location.Y, 4, true), 11) // location-y

		gsh := &GetStatsHandler{}
		statData, err := gsh.Handle(s)
		if err != nil {
			return nil, err
		}

		resp.Concat(statData)
		return resp, nil
	}

	return nil, nil
}
