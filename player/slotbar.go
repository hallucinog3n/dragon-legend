package player

import (
	"github.com/syntaxgame/dragon-legend/database"
	"github.com/syntaxgame/dragon-legend/utils"
)

type SaveSlotbarHandler struct {
}

func (h *SaveSlotbarHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	slotBar := utils.Packet{}
	slotBar.Concat(data)
	slotBar[5] = 1
	s.Character.Slotbar = slotBar
	s.Character.Update()

	resp := utils.Packet{0xAA, 0x55, 0x02, 0x00, 0xCC, 0x00, 0x55, 0xAA}
	return resp, nil
}
