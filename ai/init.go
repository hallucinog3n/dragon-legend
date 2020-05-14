package ai

import (
	"log"

	"github.com/syntaxgame/dragon-legend/database"
	"github.com/syntaxgame/dragon-legend/server"
)

func Init() {

	database.AIsByMap = make([]map[int16][]*database.AI, database.SERVER_COUNT+1)
	for s := 0; s <= database.SERVER_COUNT; s++ {
		database.AIsByMap[s] = make(map[int16][]*database.AI)
	}

	func() {
		<-server.Init

		var err error

		database.NPCPos, err = database.GetAllNPCPos()
		if err != nil {
			log.Println(err)
			return
		}

		for _, pos := range database.NPCPos {
			if pos.IsNPC && !pos.Attackable {
				server.GenerateIDForNPC(pos)
			}
		}

		database.NPCs, err = database.GetAllNPCs()
		if err != nil {
			log.Println(err)
			return
		}

		err = database.GetAllAI()
		if err != nil {
			log.Println(err)
			return
		}

		for _, ai := range database.AIs {
			database.AIsByMap[ai.Server][ai.Map] = append(database.AIsByMap[ai.Server][ai.Map], ai)
		}

		for _, AI := range database.AIs {
			if AI.ID == 0 {
				continue
			}
			pos := database.NPCPos[AI.PosID]
			npc := database.NPCs[pos.NPCID]

			AI.TargetLocation = *database.ConvertPointToLocation(AI.Coordinate)
			AI.HP = npc.MaxHp
			AI.OnSightPlayers = make(map[int]interface{})
			AI.Handler = AI.AIHandler

			if npc.Level > 200 {
				continue
			}

			server.GenerateIDForAI(AI)

			if AI.WalkingSpeed > 0 {
				go AI.Handler()
			}
		}
	}()
}
