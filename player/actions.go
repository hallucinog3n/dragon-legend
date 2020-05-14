package player

import (
	"github.com/syntaxgame/dragon-legend/database"
	"github.com/syntaxgame/dragon-legend/messaging"
	"github.com/syntaxgame/dragon-legend/nats"
	"github.com/syntaxgame/dragon-legend/utils"
)

type (
	BattleModeHandler        struct{}
	MeditationHandler        struct{}
	TargetSelectionHandler   struct{}
	TravelToCastleHandler    struct{}
	OpenTacticalSpaceHandler struct{}
	TacticalSpaceTPHandler   struct{}
	InTacticalSpaceTPHandler struct{}
	OpenLotHandler           struct{}
	EnterGateHandler         struct{}
	SendPvPRequestHandler    struct{}
	RespondPvPRequestHandler struct{}
	TransferSoulHandler      struct{}
)

var (
	FreeLotQuantities = map[int]int{10820001: 5, 10600033: 10, 10600036: 10, 17500346: 5, 10600057: 5}
	PaidLotQuantities = map[int]int{92000001: 5, 92000011: 5, 10820001: 5, 17500346: 10, 10601023: 20, 10601024: 20, 10601007: 50, 10601008: 50, 10600057: 10,
		17502966: 5, 17502967: 5, 243: 3}

	BATTLE_MODE         = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x43, 0x00, 0x55, 0xAA}
	MEDITATION_MODE     = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x82, 0x05, 0x00, 0x55, 0xAA}
	TACTICAL_SPACE_MENU = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x50, 0x01, 0x01, 0x55, 0xAA, 0xAA, 0x55, 0x05, 0x00, 0x28, 0xFF, 0x00, 0x00, 0x00, 0x55, 0xAA}
	TACTICAL_SPACE_TP   = utils.Packet{0xAA, 0x55, 0x07, 0x00, 0x01, 0xB9, 0x0A, 0x00, 0x00, 0x00, 0x01, 0x55, 0xAA}
	OPEN_LOT            = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0xA2, 0x01, 0x32, 0x00, 0x00, 0x00, 0x00, 0x01, 0x55, 0xAA}
	SELECTION_CHANGED   = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0xCF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	PVP_REQUEST         = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x2A, 0x01, 0x55, 0xAA}
	PVP_STARTED         = utils.Packet{0xAA, 0x55, 0x0A, 0x00, 0x2A, 0x02, 0x55, 0xAA}
)

func (h *BattleModeHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	battleMode := data[5]

	resp := BATTLE_MODE
	resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 5) // character pseudo id
	resp[7] = battleMode

	p := nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Type: nats.BATTLE_MODE, Data: resp}
	if err := p.Cast(); err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *MeditationHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	meditationMode := data[6] == 1
	s.Character.Meditating = meditationMode

	resp := MEDITATION_MODE
	resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 6) // character pseudo id
	resp[8] = data[6]

	p := nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Type: nats.MEDITATION_MODE, Data: resp}
	if err := p.Cast(); err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *TargetSelectionHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	id := int(utils.BytesToInt(data[5:7], true))
	s.Character.Selection = id

	resp := SELECTION_CHANGED
	resp.Insert(utils.IntToBytes(uint64(s.Character.Selection), 2, true), 5)
	return resp, nil
}

func (h *TravelToCastleHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	return s.Character.ChangeMap(233, nil)
}

func (h *OpenTacticalSpaceHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	return TACTICAL_SPACE_MENU, nil
}

func (h *TacticalSpaceTPHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	mapID := int16(data[6])
	return s.Character.ChangeMap(mapID, nil)
}

func (h *InTacticalSpaceTPHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	resp := TACTICAL_SPACE_TP
	resp[8] = data[6]
	return resp, nil
}

func (h *OpenLotHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if !s.Character.HasLot {
		return nil, nil
	}

	s.Character.HasLot = false
	paid := data[5] == 1
	dropID := 1185

	if paid && s.Character.Gold >= 150000 {
		dropID = 1186
		s.Character.Gold -= 150000
	}

	drop, ok := database.Drops[dropID]
	if drop == nil {
		return nil, nil
	}

	resp := OPEN_LOT
	itemID := 0
	for ok {
		index := 0
		seed := int(utils.RandInt(0, 1000))
		items := drop.GetItems()
		probabilities := drop.GetProbabilities()

		for _, prob := range probabilities {
			if float64(seed) > float64(prob) {
				index++
				continue
			}
			break
		}

		if index >= len(items) {
			break
		}

		itemID = items[index]
		drop, ok = database.Drops[itemID]
	}

	if itemID == 10002 {
		s.User.NCash += 1000
		go s.User.Update()

	} else {

		quantity := 1
		if paid {
			if q, ok := PaidLotQuantities[itemID]; ok {
				quantity = q
			}
		} else {
			if q, ok := FreeLotQuantities[itemID]; ok {
				quantity = q
			}
		}

		info := database.Items[int64(itemID)]
		if info.Timer > 0 {
			quantity = info.Timer
		}

		item := &database.InventorySlot{ItemID: int64(itemID), Quantity: uint(quantity)}
		r, _, err := s.Character.AddItem(item, -1, false)
		if err != nil {
			return nil, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)
	}

	resp.Insert(utils.IntToBytes(uint64(itemID), 4, true), 11) // item id
	return resp, nil
}

func (h *EnterGateHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	gateID := int(utils.BytesToInt(data[5:9], true))
	gate, ok := database.Gates[gateID]
	if !ok {
		return s.Character.ChangeMap(int16(s.Character.Map), nil)
	}

	coordinate := database.ConvertPointToLocation(gate.Point)
	return s.Character.ChangeMap(int16(gate.TargetMap), coordinate)
}

func (h *SendPvPRequestHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	pseudoID := uint16(utils.BytesToInt(data[6:8], true))
	opponent := database.FindCharacterByPseudoID(s.User.ConnectedServer, pseudoID)
	if opponent == nil {
		return nil, nil

	} else if opponent.DuelID > 0 {
		resp := messaging.SystemMessage(messaging.ALREADY_IN_PVP)
		return resp, nil
	}

	resp := PVP_REQUEST
	resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 6) // sender pseudo id

	database.GetSocket(opponent.UserID).Write(resp)
	return nil, nil
}

func (h *RespondPvPRequestHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	pseudoID := uint16(utils.BytesToInt(data[6:8], true))
	accepted := data[8] == 1

	opponent := database.FindCharacterByPseudoID(s.User.ConnectedServer, pseudoID)
	if opponent == nil {
		return nil, nil
	}

	if !accepted {
		resp := messaging.SystemMessage(messaging.PVP_REQUEST_REJECTED)
		s.Write(resp)
		database.GetSocket(opponent.UserID).Write(resp)

	} else if opponent.DuelID > 0 {
		resp := messaging.SystemMessage(messaging.ALREADY_IN_PVP)
		return resp, nil

	} else { // start pvp
		mC := database.ConvertPointToLocation(s.Character.Coordinate)
		oC := database.ConvertPointToLocation(opponent.Coordinate)
		fC := utils.Location{X: (mC.X + oC.X) / 2, Y: (mC.Y + oC.Y) / 2}

		s.Character.DuelID = opponent.ID
		opponent.DuelID = s.Character.ID

		resp := PVP_STARTED
		resp.Insert(utils.FloatToBytes(fC.X, 4, true), 6)  // flag-X
		resp.Insert(utils.FloatToBytes(fC.Y, 4, true), 10) // flag-Y

		//p := nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Type: nats.PVP_START, Data: resp}
		//p.Cast()

		s.Character.Socket.Write(resp)
		opponent.Socket.Write(resp)

		go s.Character.StartPvP(3)
		go opponent.StartPvP(3)
	}

	return nil, nil
}

func (h *TransferSoulHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	//pseudoID := uint16(utils.BytesToInt(data[6:8], true))
	resp := utils.Packet{0xAA, 0x55, 0x06, 0x00, 0xA5, 0x01, 0x0A, 0x00, 0x55, 0xAA}
	resp.Insert(data[6:8], 8)
	resp.Print()
	return resp, nil
}
