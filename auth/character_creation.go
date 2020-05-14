package auth

import (
	"github.com/syntaxgame/dragon-legend/database"
	"github.com/syntaxgame/dragon-legend/logging"
	"github.com/syntaxgame/dragon-legend/messaging"
	"github.com/syntaxgame/dragon-legend/utils"
)

type CancelCharacterCreationHandler struct {
}

type CharacterCreationHandler struct {
	characterType int
	faction       int
	height        int
	name          string
}

var (
	CHARACTER_CREATED = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x01, 0x03, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA}
)

func (ccch *CancelCharacterCreationHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	lch := &ListCharactersHandler{}
	return lch.showCharacterMenu(s)
}

func (cch *CharacterCreationHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	index := 7
	length := int(data[index])
	index += 1

	cch.name = string(data[8 : length+8])
	index += len(cch.name)

	cch.characterType = int(data[index])
	index += 1

	if cch.characterType == 52 { // Monk creation
		return messaging.SystemMessage(messaging.INCORRECT_REGISTRATION), nil
	}

	characters, err := database.FindCharactersByUserID(s.User.ID)
	if err != nil {
		return nil, err
	}

	if len(characters) > 0 {
		cch.faction = characters[0].Faction
	} else {
		cch.faction = int(data[index])
	}
	index += 1

	cch.height = int(data[index])
	index += 1

	// TODO => FACE AND HEAD

	return cch.createCharacter(s)
}

func (cch *CharacterCreationHandler) createCharacter(s *database.Socket) ([]byte, error) {

	ok, err := database.IsValidUsername(cch.name)
	if err != nil {
		return nil, err
	} else if !ok {
		return messaging.SystemMessage(messaging.INVALID_NAME), nil
	} else if cch.faction == 0 {
		return messaging.SystemMessage(messaging.EMPTY_FACTION), nil
	}

	coordinate := database.SavePoints[1]
	if err != nil {
		return nil, err
	}

	character := &database.Character{
		Type:           cch.characterType,
		UserID:         s.User.ID,
		Name:           cch.name,
		Epoch:          0,
		Faction:        cch.faction,
		Height:         cch.height,
		Level:          1,
		Class:          0,
		IsOnline:       false,
		IsActive:       false,
		Gold:           0,
		Map:            1,
		Exp:            0,
		HTVisibility:   0,
		WeaponSlot:     3,
		RunningSpeed:   5.6,
		GuildID:        -1,
		ExpMultiplier:  1,
		DropMultiplier: 1,
		Slotbar:        []byte{},
		Coordinate:     coordinate.Point,
		AidTime:        7200,
	}

	err = character.Create()
	if err != nil {
		return nil, err
	}

	character.AddItem(&database.InventorySlot{ItemID: 17200576, Quantity: 1}, -1, false)
	character.AddItem(&database.InventorySlot{ItemID: 17500335, Quantity: 1}, -1, false)
	character.Update()

	stat := &database.Stat{}
	err = stat.Create(character)
	if err != nil {
		return nil, err
	}

	skills := &database.Skills{}
	err = skills.Create(character)
	if err != nil {
		return nil, err
	}

	stat, err = database.FindStatByID(character.ID)
	if err != nil {
		return nil, err
	}

	err = stat.Calculate()
	if err != nil {
		return nil, err
	}

	resp := CHARACTER_CREATED
	length := int16(len(cch.name)) + 10
	resp.SetLength(length)

	resp.Insert(utils.IntToBytes(uint64(character.ID), 4, true), 9) // character id

	resp[13] = byte(len(cch.name)) // character name length

	resp.Insert([]byte(cch.name), 14) // character name

	lch := &ListCharactersHandler{}
	data, err := lch.showCharacterMenu(s)
	if err != nil {
		return nil, err
	}

	logger.Log(logging.ACTION_CREATE_CHARACTER, character.ID, "Character created", s.User.ID)
	resp.Concat(data)
	return resp, nil
}
