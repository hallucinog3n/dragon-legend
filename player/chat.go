package player

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/syntaxgame/dragon-legend/database"
	"github.com/syntaxgame/dragon-legend/messaging"
	"github.com/syntaxgame/dragon-legend/nats"
	"github.com/syntaxgame/dragon-legend/server"
	"github.com/syntaxgame/dragon-legend/utils"
	"gopkg.in/guregu/null.v3"

	"github.com/thoas/go-funk"
)

type ChatHandler struct {
	chatType  int64
	message   string
	receivers map[int]*database.Character
}

var (
	CHAT_MESSAGE  = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x00, 0x55, 0xAA}
	SHOUT_MESSAGE = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x71, 0x0E, 0x00, 0x00, 0x55, 0xAA}
	ANNOUNCEMENT  = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x71, 0x06, 0x00, 0x55, 0xAA}
)

func (h *ChatHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.Character == nil {
		return nil, nil
	}

	user, err := database.FindUserByID(s.Character.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	stat := s.Stats
	if stat == nil {
		return nil, nil
	}

	h.chatType = utils.BytesToInt(data[4:6], false)

	switch h.chatType {
	case 28929: // normal chat
		messageLen := utils.BytesToInt(data[6:8], true)
		h.message = string(data[8 : messageLen+8])

		return h.normalChat(s)
	case 28930: // private chat
		index := 6
		recNameLength := int(data[index])
		index++

		recName := string(data[index : index+recNameLength])
		index += recNameLength

		c, err := database.FindCharacterByName(recName)
		if err != nil {
			return nil, err
		} else if c == nil {
			return messaging.SystemMessage(messaging.WHISPER_FAILED), nil
		}

		h.receivers = map[int]*database.Character{c.ID: c}

		messageLen := int(utils.BytesToInt(data[index:index+2], true))
		index += 2

		h.message = string(data[index : index+messageLen])
		return h.chatWithReceivers(s, h.createChatMessage)

	case 28931: // party chat
		party := database.FindParty(s.Character)
		if party == nil {
			return nil, nil
		}

		messageLen := int(utils.BytesToInt(data[6:8], true))
		h.message = string(data[8 : messageLen+8])

		members := funk.Filter(party.GetMembers(), func(m *database.PartyMember) bool {
			return m.Accepted
		}).([]*database.PartyMember)
		members = append(members, &database.PartyMember{Character: party.Leader, Accepted: true})

		h.receivers = map[int]*database.Character{}
		for _, m := range members {
			if m.ID == s.Character.ID {
				continue
			}

			h.receivers[m.ID] = m.Character
		}

		return h.chatWithReceivers(s, h.createChatMessage)

	case 28932: // guild chat
		if s.Character.GuildID > 0 {
			guild, err := database.FindGuildByID(s.Character.GuildID)
			if err != nil {
				return nil, err
			}

			members, err := guild.GetMembers()
			if err != nil {
				return nil, err
			}

			messageLen := int(utils.BytesToInt(data[6:8], true))
			h.message = string(data[8 : messageLen+8])
			h.receivers = map[int]*database.Character{}

			for _, m := range members {
				c, err := database.FindCharacterByID(m.ID)
				if err != nil || c == nil || !c.IsOnline || c.ID == s.Character.ID {
					continue
				}

				h.receivers[m.ID] = c
			}

			return h.chatWithReceivers(s, h.createChatMessage)
		}

	case 28933, 28946: // roar chat
		if stat.CHI < 100 || time.Now().Sub(s.Character.LastRoar) < 10*time.Second {
			return nil, nil
		}

		s.Character.LastRoar = time.Now()
		characters, err := database.FindCharactersInServer(user.ConnectedServer)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		//delete(characters, s.Character.ID)
		h.receivers = characters

		stat.CHI -= 100

		index := 6
		messageLen := int(utils.BytesToInt(data[index:index+2], true))
		index += 2

		h.message = string(data[index : index+messageLen])

		resp := utils.Packet{}
		_, err = h.chatWithReceivers(s, h.createChatMessage)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		//resp.Concat(chat)
		resp.Concat(s.Character.GetHPandChi())
		return resp, nil

	case 28935: // commands
		index := 6
		messageLen := int(data[index])
		index++

		h.message = string(data[index : index+messageLen])
		return h.cmdMessage(s, data)

	case 28943: // shout
		return h.Shout(s, data)

	case 28945: // faction chat
		characters, err := database.FindCharactersInServer(user.ConnectedServer)
		if err != nil {
			return nil, err
		}

		//delete(characters, s.Character.ID)
		for _, c := range characters {
			if c.Faction != s.Character.Faction {
				delete(characters, c.ID)
			}
		}

		h.receivers = characters
		index := 6
		messageLen := int(utils.BytesToInt(data[index:index+2], true))
		index += 2

		h.message = string(data[index : index+messageLen])
		resp := utils.Packet{}
		_, err = h.chatWithReceivers(s, h.createChatMessage)
		if err != nil {
			return nil, err
		}

		//resp.Concat(chat)
		return resp, nil

	}

	return nil, nil
}

func (h *ChatHandler) Shout(s *database.Socket, data []byte) ([]byte, error) {
	if time.Now().Sub(s.Character.LastRoar) < 10*time.Second {
		return nil, nil
	}

	characters, err := database.FindOnlineCharacters()
	if err != nil {
		log.Println(err)
		return nil, err
	}

	//delete(characters, s.Character.ID)

	slot, _, err := s.Character.FindItemInInventory(nil, 15900001, 17500181, 17502689, 13000131)
	if err != nil {
		log.Println(err)
		return nil, err
	} else if slot == -1 {
		return nil, nil
	}

	resp := s.Character.DecrementItem(slot, 1)

	index := 6
	messageLen := int(data[index])
	index++

	h.chatType = 28942
	h.receivers = characters
	h.message = string(data[index : index+messageLen])

	_, err = h.chatWithReceivers(s, h.createShoutMessage)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	//resp.Concat(chat)
	return *resp, nil
}

func (h *ChatHandler) createChatMessage(s *database.Socket) *utils.Packet {

	resp := CHAT_MESSAGE

	index := 4
	resp.Insert(utils.IntToBytes(uint64(h.chatType), 2, false), index) // chat type
	index += 2

	if h.chatType != 28946 {
		resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), index) // sender character pseudo id
		index += 2
	}

	resp[index] = byte(len(s.Character.Name)) // character name length
	index++

	resp.Insert([]byte(s.Character.Name), index) // character name
	index += len(s.Character.Name)

	resp.Insert(utils.IntToBytes(uint64(len(h.message)), 2, true), index) // message length
	index += 2

	resp.Insert([]byte(h.message), index) // message
	index += len(h.message)

	length := index - 4
	resp.SetLength(int16(length)) // packet length

	return &resp
}

func (h *ChatHandler) createShoutMessage(s *database.Socket) *utils.Packet {

	resp := SHOUT_MESSAGE
	length := len(s.Character.Name) + len(h.message) + 6
	resp.SetLength(int16(length)) // packet length

	index := 4
	resp.Insert(utils.IntToBytes(uint64(h.chatType), 2, false), index) // chat type
	index += 2

	resp[index] = byte(len(s.Character.Name)) // character name length
	index++

	resp.Insert([]byte(s.Character.Name), index) // character name
	index += len(s.Character.Name)

	resp[index] = byte(len(h.message)) // message length
	index++

	resp.Insert([]byte(h.message), index) // message
	return &resp
}

func (h *ChatHandler) normalChat(s *database.Socket) ([]byte, error) {

	if _, ok := server.MutedPlayers.Get(s.User.ID); ok {
		msg := "Chatting with this account is prohibited. Please contact our customer support service for more information."
		return messaging.InfoMessage(msg), nil
	}

	resp := h.createChatMessage(s)
	p := &nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Data: *resp, Type: nats.CHAT_NORMAL}
	err := p.Cast()

	return nil, err
}

func (h *ChatHandler) chatWithReceivers(s *database.Socket, msgHandler func(*database.Socket) *utils.Packet) ([]byte, error) {

	if _, ok := server.MutedPlayers.Get(s.User.ID); ok {
		msg := "Chatting with this account is prohibited. Please contact our customer support service for more information."
		return messaging.InfoMessage(msg), nil
	}

	resp := msgHandler(s)

	for _, c := range h.receivers {
		if c == nil || !c.IsOnline {
			if h.chatType == 28930 { // PM
				return messaging.SystemMessage(messaging.WHISPER_FAILED), nil
			}
			continue
		}

		socket := database.GetSocket(c.UserID)
		if socket != nil {
			_, err := socket.Conn.Write(*resp)
			if err != nil {
				log.Println(err)
				return nil, err
			}
		}
	}

	return *resp, nil
}

func makeAnnouncement(msg string) {
	length := int16(len(msg) + 3)

	resp := ANNOUNCEMENT
	resp.SetLength(length)
	resp[6] = byte(len(msg))
	resp.Insert([]byte(msg), 7)

	p := nats.CastPacket{CastNear: false, Data: resp}
	p.Cast()
}

func (h *ChatHandler) cmdMessage(s *database.Socket, data []byte) ([]byte, error) {

	var (
		err  error
		resp utils.Packet
	)

	if parts := strings.Split(h.message, " "); len(parts) > 0 {
		cmd := strings.ToLower(strings.TrimPrefix(parts[0], "/"))
		switch cmd {
		case "shout":
			return h.Shout(s, data)

		case "announce":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			msg := strings.Join(parts[1:], " ")
			makeAnnouncement(msg)

		case "item":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			itemID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			quantity := int64(1)
			if len(parts) >= 3 {
				quantity, err = strconv.ParseInt(parts[2], 10, 64)
				if err != nil {
					return nil, err
				}
			}

			ch := s.Character
			if len(parts) >= 4 {
				chID, err := strconv.ParseInt(parts[3], 10, 64)
				if err == nil {
					chr, err := database.FindCharacterByID(int(chID))
					if err == nil {
						ch = chr
					}
				}
			}

			item := &database.InventorySlot{ItemID: itemID, Quantity: uint(quantity)}
			info := database.Items[itemID]

			if info.GetType() == database.PET_TYPE {
				petInfo := database.Pets[itemID]
				expInfo := database.PetExps[petInfo.Level-1]

				item.Pet = &database.PetSlot{
					Fullness: 100, Loyalty: 100,
					Exp:   uint64(expInfo.ReqExpEvo1),
					HP:    petInfo.BaseHP,
					Level: byte(petInfo.Level),
					Name:  petInfo.Name,
					CHI:   petInfo.BaseChi}
			}

			r, _, err := ch.AddItem(item, -1, false)
			if err != nil {
				return nil, err
			}

			ch.Socket.Write(*r)
			return nil, nil

		case "gold":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			amount, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			s.Character.Gold += uint64(amount)
			h := &GetGoldHandler{}

			return h.Handle(s)

		case "upgrade":
			if s.User.UserType < server.GM_USER || len(parts) < 3 {
				return nil, nil
			}

			slots, err := s.Character.InventorySlots()
			if err != nil {
				return nil, err
			}

			slotID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			code, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, err
			}

			count := int64(1)
			if len(parts) > 3 {
				count, err = strconv.ParseInt(parts[3], 10, 64)
				if err != nil {
					return nil, err
				}
			}

			codes := []byte{}
			for i := 0; i < int(count); i++ {
				codes = append(codes, byte(code))
			}

			item := slots[slotID]
			return item.Upgrade(int16(slotID), codes...), nil

		case "exp":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			amount, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			ch := s.Character
			if len(parts) > 2 {
				chID, err := strconv.ParseInt(parts[2], 10, 64)
				if err == nil {
					chr, err := database.FindCharacterByID(int(chID))
					if err == nil {
						ch = chr
					}
				}
			}

			data, levelUp := ch.AddExp(amount)
			if levelUp {
				statData, err := ch.GetStats()
				if err == nil && ch.Socket != nil {
					ch.Socket.Write(statData)
				}
			}

			if ch.Socket != nil {
				ch.Socket.Write(data)
			}

			return nil, nil

		case "home":
			data, err := s.Character.ChangeMap(s.Character.Map, nil)
			if err != nil {
				return nil, err
			}

			resp.Concat(data)

		case "map":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			mapID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			if len(parts) >= 3 {
				c, err := database.FindCharacterByName(parts[2])
				if err != nil {
					return nil, err
				}

				data, err := c.ChangeMap(int16(mapID), nil)
				if err != nil {
					return nil, err
				}

				database.GetSocket(c.UserID).Write(data)
				return nil, nil
			}

			return s.Character.ChangeMap(int16(mapID), nil)

		case "cash":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			amount, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			userID := parts[2]
			user, err := database.FindUserByID(userID)
			if err != nil {
				return nil, err
			} else if user == nil {
				return nil, nil
			}

			user.NCash += uint64(amount)
			user.Update()

			return messaging.InfoMessage(fmt.Sprintf("%d nCash loaded to %s (%s).", amount, user.Username, user.ID)), nil

		case "mob":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			posId, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			npcPos := database.NPCPos[int(posId)]
			npc, ok := database.NPCs[npcPos.NPCID]
			if !ok {
				return nil, nil
			}

			ai := &database.AI{ID: len(database.AIs), HP: npc.MaxHp, Map: npcPos.MapID, PosID: npcPos.ID, RunningSpeed: 10, Server: 1, WalkingSpeed: 5, Once: true}
			server.GenerateIDForAI(ai)
			ai.OnSightPlayers = make(map[int]interface{})

			minLoc := database.ConvertPointToLocation(npcPos.MinLocation)
			maxLoc := database.ConvertPointToLocation(npcPos.MaxLocation)
			loc := utils.Location{X: utils.RandFloat(minLoc.X, maxLoc.X), Y: utils.RandFloat(minLoc.Y, maxLoc.Y)}

			ai.Coordinate = loc.String()
			fmt.Println(ai.Coordinate)
			ai.Handler = ai.AIHandler
			go ai.Handler()

			makeAnnouncement(fmt.Sprintf("%s has been roaring.", npc.Name))

			database.AIsByMap[ai.Server][npcPos.MapID] = append(database.AIsByMap[ai.Server][npcPos.MapID], ai)
			database.AIs[ai.ID] = ai

		case "relic":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			itemID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			ch := s.Character
			if len(parts) >= 3 {
				chID, err := strconv.ParseInt(parts[2], 10, 64)
				if err == nil {
					chr, err := database.FindCharacterByID(int(chID))
					if err == nil {
						ch = chr
					}
				}
			}

			slot, err := ch.FindFreeSlot()
			if err != nil {
				return nil, nil
			}

			itemData, _, _ := ch.AddItem(&database.InventorySlot{ItemID: itemID, Quantity: 1}, slot, true)
			if itemData != nil {
				ch.Socket.Write(*itemData)

				relicDrop := ch.RelicDrop(int64(itemID))
				p := nats.CastPacket{CastNear: false, Data: relicDrop, Type: nats.ITEM_DROP}
				p.Cast()
			}

		case "main":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}

			countMaintenance(60)

		case "ban":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			userID := parts[1]
			user, err := database.FindUserByID(userID)
			if err != nil {
				return nil, err
			} else if user == nil {
				return nil, nil
			}

			hours, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, err
			}

			user.UserType = 0
			user.DisabledUntil = null.NewTime(time.Now().Add(time.Hour*time.Duration(hours)), true)
			user.Update()

			database.GetSocket(userID).Conn.Close()

		case "mute":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			dumb, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}

			server.MutedPlayers.Set(dumb.UserID, struct{}{})

		case "unmute":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			dumb, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}

			server.MutedPlayers.Remove(dumb.UserID)

		case "uid":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			} else if c == nil {
				return nil, nil
			}

			resp = messaging.InfoMessage(c.UserID)

		case "uuid":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			user, err := database.FindUserByName(parts[1])
			if err != nil {
				return nil, err
			} else if user == nil {
				return nil, nil
			}

			resp = messaging.InfoMessage(user.ID)

		case "visibility":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			s.Character.Invisible = parts[1] == "1"

		case "kick":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			dumb, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}

			database.GetSocket(dumb.UserID).Conn.Close()

		case "tp":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			x, err := strconv.ParseFloat(parts[1], 10)
			if err != nil {
				return nil, err
			}

			y, err := strconv.ParseFloat(parts[2], 10)
			if err != nil {
				return nil, err
			}

			return s.Character.Teleport(database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))), nil

		case "tpp":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}

			return s.Character.ChangeMap(c.Map, database.ConvertPointToLocation(c.Coordinate))

		case "speed":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			speed, err := strconv.ParseFloat(parts[1], 10)
			if err != nil {
				return nil, err
			}

			s.Character.RunningSpeed = speed

		case "online":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}

			characters, err := database.FindOnlineCharacters()
			if err != nil {
				return nil, err
			}

			online := funk.Values(characters).([]*database.Character)
			sort.Slice(online, func(i, j int) bool {
				return online[i].Name < online[j].Name
			})

			resp.Concat(messaging.InfoMessage(fmt.Sprintf("%d player(s) online.", len(characters))))

			for _, c := range online {
				u, _ := database.FindUserByID(c.UserID)
				if u == nil {
					continue
				}

				resp.Concat(messaging.InfoMessage(fmt.Sprintf("%s is in map %d (Dragon%d) at %s.", c.Name, c.Map, u.ConnectedServer, c.Coordinate)))
			}

		case "name":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			id, _ := strconv.Atoi(parts[1])
			c, err := database.FindCharacterByID(int(id))
			if err != nil {
				return nil, err
			}

			c2, err := database.FindCharacterByName(parts[2])
			if err != nil {
				return nil, err
			} else if c2 != nil {
				return nil, nil
			}

			c.Name = parts[2]
			c.Update()

		case "role":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			id, _ := strconv.Atoi(parts[1])
			c, err := database.FindCharacterByID(int(id))
			if err != nil {
				return nil, err
			}

			user, err := database.FindUserByID(c.UserID)
			if err != nil {
				return nil, err
			}

			role, err := strconv.Atoi(parts[2])
			if err != nil {
				return nil, err
			}

			user.UserType = int8(role)
			user.Update()

		case "type":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			id, _ := strconv.Atoi(parts[1])
			c, err := database.FindCharacterByID(int(id))
			if err != nil {
				return nil, err
			}

			t, err := strconv.Atoi(parts[2])
			if err != nil {
				return nil, err
			}

			c.Type = t
			c.Update()
		}

	}

	return resp, err
}

func countMaintenance(cd int) {
	msg := fmt.Sprintf("There will be maintenance after %d seconds. Please log out in order to prevent any inconvenience.", cd)
	makeAnnouncement(msg)

	if cd > 0 {
		time.AfterFunc(time.Second*10, func() {
			countMaintenance(cd - 10)
		})
	} else {
		//os.Exit(0)
	}
}
