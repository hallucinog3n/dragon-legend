package npc

import (
	"fmt"
	"strings"

	"github.com/syntaxgame/dragon-legend/database"
	"github.com/syntaxgame/dragon-legend/utils"

	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
)

type OpenHandler struct {
}

type PressButtonHandler struct {
}

var (
	shops = map[int]int{20002: 7, 20003: 2, 20004: 4, 20005: 1, 20009: 8, 20010: 10, 20011: 10, 20013: 25,
		20024: 6, 20025: 6, 20026: 11, 20033: 21, 20034: 22, 20035: 23, 20036: 24, 20044: 21, 20047: 21, 20082: 21,
		20083: 21, 20084: 21, 20085: 23, 20086: 22, 20087: 21, 20094: 103, 20095: 100, 20105: 21, 20133: 21,
		20146: 21, 20151: 6, 20173: 327, 20211: 25, 20239: 21, 20415: 21}

	COMPOSITION_MENU    = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x0F, 0x01, 0x55, 0xAA}
	OPEN_SHOP           = utils.Packet{0xAA, 0x55, 0x07, 0x00, 0x57, 0x03, 0x01, 0x55, 0xAA}
	NPC_MENU            = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x57, 0x02, 0x55, 0xAA}
	STRENGTHEN_MENU     = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x08, 0x01, 0x55, 0xAA}
	JOB_PROMOTED        = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x09, 0x00, 0x55, 0xAA}
	NOT_ENOUGH_LEVEL    = utils.Packet{0xAA, 0x55, 0x0B, 0x00, 0x57, 0x02, 0x38, 0x42, 0x0F, 0x00, 0x00, 0x55, 0xAA}
	INVALID_CLASS       = utils.Packet{0xAA, 0x55, 0x0B, 0x00, 0x57, 0x02, 0x49, 0x2F, 0x00, 0x00, 0x00, 0x55, 0xAA}
	GUILD_MENU          = utils.Packet{0xAA, 0x55, 0x02, 0x00, 0x57, 0x0D, 0x55, 0xAA}
	DISMANTLE_MENU      = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x16, 0x01, 0x55, 0xAA}
	EXTRACTION_MENU     = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x17, 0x01, 0x55, 0xAA}
	ADV_FUSION_MENU     = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x32, 0x01, 0x55, 0xAA}
	TACTICAL_SPACE      = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x50, 0x01, 0x01, 0x55, 0xAA}
	CREATE_SOCKET_MENU  = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x39, 0x01, 0x55, 0xAA}
	UPGRADE_SOCKET_MENU = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x3A, 0x01, 0x55, 0xAA}
	CONSIGNMENT_MENU    = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x42, 0x01, 0x55, 0xAA}
)

func (h *OpenHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	if c == nil {
		return nil, nil
	}

	u := s.User
	if u == nil {
		return nil, nil
	}

	id := uint16(utils.BytesToInt(data[6:10], true))
	pos, ok := database.GetFromRegister(1, c.Map, id).(*database.NpcPosition)
	if !ok {
		return nil, nil
	}

	npc := database.NPCs[pos.NPCID]

	if npc.ID == 20147 { // Ice Palace Mistress Lord
		coordinate := &utils.Location{X: 163, Y: 350}
		return c.Teleport(coordinate), nil

	} else if npc.ID == 20055 { // Mysterious Tombstone
		coordinate := &utils.Location{X: 365, Y: 477}
		return c.Teleport(coordinate), nil

	} else if npc.ID == 20056 { // Mysterious Tombstone (R)
		coordinate := &utils.Location{X: 70, Y: 450}
		return c.Teleport(coordinate), nil

	} else if npc.ID == 22351 { // Golden Castle Teleport Tombstone
		return c.ChangeMap(236, nil)

	} else if npc.ID == 22357 { // 2nd FL Entrance
		return c.ChangeMap(237, nil)

	} else if npc.ID == 22358 { // 3rd FL Entrance
		return c.ChangeMap(239, nil)
	}

	npcScript := database.NPCScripts[npc.ID]
	if npcScript == nil {
		return nil, nil
	}

	script := string(npcScript.Script)
	textID := gjson.Get(script, "text").Int()
	actions := []int{}

	for _, action := range gjson.Get(script, "actions").Array() {
		actions = append(actions, int(action.Int()))
	}

	resp := NPC_MENU
	resp.Insert(utils.IntToBytes(uint64(npc.ID), 4, true), 6)        // npc id
	resp.Insert(utils.IntToBytes(uint64(textID), 4, true), 10)       // text id
	resp.Insert(utils.IntToBytes(uint64(len(actions)), 1, true), 14) // action length

	index, length := 15, int16(11)
	for i, action := range actions {
		resp.Insert(utils.IntToBytes(uint64(action), 4, true), index) // action
		index += 4

		resp.Insert(utils.IntToBytes(uint64(npc.ID), 2, true), index) // npc id
		index += 2

		resp.Insert(utils.IntToBytes(uint64(i+1), 2, true), index) // action index
		index += 2

		length += 8
	}

	resp.SetLength(length)
	return resp, nil
}

func (h *PressButtonHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	if c == nil {
		return nil, nil
	}

	npcID := int(utils.BytesToInt(data[6:8], true))
	index := int(utils.BytesToInt(data[8:10], true))
	indexes := []int{index & 7, (index & 56) / 8, (index & 448) / 64, (index & 3584) / 512, (index & 28672) / 4096}
	indexes = funk.FilterInt(indexes, func(i int) bool {
		return i > 0
	})

	npcScript := database.NPCScripts[npcID]
	if npcScript == nil {
		return nil, nil
	}

	script := string(npcScript.Script)
	key := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(indexes)), "."), "[]")

	script = gjson.Get(script, key).String()
	if script != "" {
		textID := int(gjson.Get(script, fmt.Sprintf("text")).Int())
		actions := []int{}

		for _, action := range gjson.Get(script, "actions").Array() {
			actions = append(actions, int(action.Int()))
		}

		resp := GetNPCMenu(npcID, textID, index, actions)
		return resp, nil
	} else { // Action button

		key := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(indexes[:len(indexes)-1])), "."), "[]")
		script = string(npcScript.Script)
		if key != "" {
			script = gjson.Get(script, key).String()
		}

		actions := gjson.Get(script, "actions").Array()
		actIndex := indexes[len(indexes)-1] - 1
		actID := actions[actIndex].Int()

		resp := utils.Packet{}

		var err error
		book1, book2, job := 0, 0, 0
		switch actID {
		case 1: // Exchange
			shopNo := shops[npcID]
			resp = OPEN_SHOP
			resp.Insert(utils.IntToBytes(uint64(shopNo), 4, true), 7) // shop id

		case 2: // Compositon
			resp = COMPOSITION_MENU

		case 4: // Strengthen
			resp = STRENGTHEN_MENU

		case 6: // Deposit
			resp = c.BankItems()

		case 13: // Accept
			switch npcID {
			case 20006: // Hunter trainer
				book1, job = 16210003, 13
				resp, err = firstJobPromotion(c, book1, job, npcID)
				if err != nil {
					return nil, err
				}
			case 20020: // Warrior trainer
				book1, job = 16210001, 11
				resp, err = firstJobPromotion(c, book1, job, npcID)
				if err != nil {
					return nil, err
				}
			case 20021: // Physician trainer
				book1, job = 16210002, 12
				resp, err = firstJobPromotion(c, book1, job, npcID)
				if err != nil {
					return nil, err
				}
			case 20022: // Assassin trainer
				book1, job = 16210004, 14
				resp, err = firstJobPromotion(c, book1, job, npcID)
				if err != nil {
					return nil, err
				}

			case 20415: // RDL tavern
				resp, _ = c.ChangeMap(254, nil)
			}

		case 64: // Create Guild
			if c.GuildID == -1 {
				resp = GUILD_MENU
			}

		case 77: // Move to Souther Plains
			resp, _ = c.ChangeMap(7, nil)

		case 78: // Move to Dragon Castle
			resp, _ = c.ChangeMap(1, nil)

		case 86: // Move to Spirit Spire
			resp, _ = c.ChangeMap(5, nil)

		case 103: // Move to Highlands
			resp, _ = c.ChangeMap(2, nil)

		case 104: // Move to Venom Swamp
			resp, _ = c.ChangeMap(3, nil)

		case 106: // Move to Silent Valley
			resp, _ = c.ChangeMap(11, nil)

		case 148: // Become a Champion
			book1, book2, job = 16100039, 16100200, 21
			resp, err = secondJobPromotion(c, book1, book2, 11, job, npcID)
			if err != nil {
				return nil, err
			}
		case 149: // Become a Musa
			book1, book2, job = 16100040, 16100200, 22
			resp, err = secondJobPromotion(c, book1, book2, 11, job, npcID)
			if err != nil {
				return nil, err
			}
		case 151: // Become a Surgeon
			book1, book2, job = 16100041, 16100200, 23
			resp, err = secondJobPromotion(c, book1, book2, 12, job, npcID)
			if err != nil {
				return nil, err
			}
		case 152: // Become a Combat Medic
			book1, book2, job = 16100042, 16100200, 24
			resp, err = secondJobPromotion(c, book1, book2, 12, job, npcID)
			if err != nil {
				return nil, err
			}
		case 154: // Become a Slayer
			book1, book2, job = 16100043, 16100200, 27
			resp, err = secondJobPromotion(c, book1, book2, 14, job, npcID)
			if err != nil {
				return nil, err
			}
		case 155: // Become a Shinobi
			book1, book2, job = 16100044, 16100200, 28
			resp, err = secondJobPromotion(c, book1, book2, 14, job, npcID)
			if err != nil {
				return nil, err
			}
		case 157: // Become a Tracker
			book1, book2, job = 16100045, 16100200, 25
			resp, err = secondJobPromotion(c, book1, book2, 13, job, npcID)
			if err != nil {
				return nil, err
			}
		case 158: // Become a Ranger
			book1, book2, job = 16100046, 16100200, 26
			resp, err = secondJobPromotion(c, book1, book2, 13, job, npcID)
			if err != nil {
				return nil, err
			}

		case 194: // Dismantle
			resp = DISMANTLE_MENU

		case 195: // Extraction
			resp = EXTRACTION_MENU

		case 524: // Exit Paid Zone
			if maps, ok := database.DKMaps[c.Map]; ok {
				resp, err = c.ChangeMap(maps[0], nil)
				if err != nil {
					return nil, err
				}
			}

		case 525: // Enter Paid Zone
			f := func(item *database.InventorySlot) bool {
				return item.Activated
			}
			_, item, err := c.FindItemInInventory(f, 15700040, 15710087)
			if err != nil {
				return nil, err
			} else if item == nil { // You don't have ticket
				resp := GetNPCMenu(npcID, 999993, 0, nil)
				return resp, nil
			}

			if maps, ok := database.DKMaps[c.Map]; ok {
				resp, err = c.ChangeMap(maps[1], nil)
				if err != nil {
					return nil, err
				}
			}

		case 559: // Advanced Fusion
			resp = ADV_FUSION_MENU

		case 631: // Tactical Space
			resp = TACTICAL_SPACE

		case 732: // Flexible Castle Entry
			f := func(item *database.InventorySlot) bool {
				return item.Activated
			}
			_, item, err := c.FindItemInInventory(f, 15710087)
			if err != nil {
				return nil, err
			} else if item == nil { // You don't have ticket
				resp := GetNPCMenu(npcID, 999993, 0, nil)
				return resp, nil
			}

			if maps, ok := database.DKMaps[c.Map]; ok {
				resp, err = c.ChangeMap(maps[2], nil)
				if err != nil {
					return nil, err
				}
			}

		case 737: // Create Socket
			//resp = CREATE_SOCKET_MENU

		case 738: // Upgrade Socket
			//resp = UPGRADE_SOCKET_MENU

		case 970: // Consignment
			resp = CONSIGNMENT_MENU

		case 3306: // Aid 2hr
			_, item, err := c.FindItemInInventory(nil, 13000170)
			if item != nil || err != nil {
				return nil, nil
			}

			cost := 100 * c.Level * c.Level * c.Level
			cost /= 2

			if c.Gold < uint64(cost) {
				return nil, nil
			}

			itemData, _, err := c.AddItem(&database.InventorySlot{ItemID: 13000170, Quantity: 120}, -1, true)
			if err != nil {
				return nil, err
			}

			c.LootGold(-uint64(cost))
			resp.Concat(*itemData)
			resp.Concat(c.GetGold())

		case 3307: // Aid 4hr
			_, item, err := c.FindItemInInventory(nil, 13000171)
			if item != nil || err != nil {
				return nil, nil
			}

			cost := 100 * c.Level * c.Level * c.Level * 3
			cost /= 2

			if c.Gold < uint64(cost) {
				return nil, nil
			}

			itemData, _, err := c.AddItem(&database.InventorySlot{ItemID: 13000171, Quantity: 240}, -1, true)
			if err != nil {
				return nil, err
			}

			c.LootGold(-uint64(cost))
			resp.Concat(*itemData)
			resp.Concat(c.GetGold())

		case 197101: // Move to Marketplace
			resp, _ = c.ChangeMap(254, nil)
		}

		return resp, nil
	}
}

func GetNPCMenu(npcID, textID, index int, actions []int) []byte {
	resp := NPC_MENU
	resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6)         // npc id
	resp.Insert(utils.IntToBytes(uint64(textID), 4, true), 10)       // text id
	resp.Insert(utils.IntToBytes(uint64(len(actions)), 1, true), 14) // action length

	counter, length := 15, int16(11)
	for i, action := range actions {
		resp.Insert(utils.IntToBytes(uint64(action), 4, true), counter) // action
		counter += 4

		resp.Insert(utils.IntToBytes(uint64(npcID), 2, true), counter) // npc id
		counter += 2

		actIndex := int(index) + (i+1)<<(len(actions)*3)
		resp.Insert(utils.IntToBytes(uint64(actIndex), 2, true), counter) // action index
		counter += 2

		length += 8
	}

	resp.SetLength(length)
	return resp
}

func firstJobPromotion(c *database.Character, book, job, npcID int) (utils.Packet, error) {
	resp := utils.Packet{}
	if c.Class == 0 && c.Level >= 10 {
		c.Class = job
		resp = JOB_PROMOTED
		resp[6] = byte(job)

		r, _, err := c.AddItem(&database.InventorySlot{ItemID: int64(book), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)

	} else if c.Level < 10 {
		resp = NOT_ENOUGH_LEVEL
		resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6) // npc id
	} else {
		resp = INVALID_CLASS
		resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6) // npc id
	}

	return resp, nil
}

func secondJobPromotion(c *database.Character, book1, book2, preJob, job, npcID int) (utils.Packet, error) {
	resp := utils.Packet{}
	if c.Class == preJob && c.Level >= 50 {
		c.Class = job
		resp = JOB_PROMOTED
		resp[6] = byte(job)

		r, _, err := c.AddItem(&database.InventorySlot{ItemID: int64(book1), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)

		r, _, err = c.AddItem(&database.InventorySlot{ItemID: int64(book2), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)

	} else if c.Level < 50 {
		resp := NOT_ENOUGH_LEVEL
		resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6) // npc id
	} else {
		resp = INVALID_CLASS
		resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6) // npc id
	}

	return resp, nil
}
