package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/robfig/cron"
	"github.com/syntaxgame/dragon-legend/ai"
	_ "github.com/syntaxgame/dragon-legend/ai"
	"github.com/syntaxgame/dragon-legend/api"
	"github.com/syntaxgame/dragon-legend/config"
	"github.com/syntaxgame/dragon-legend/database"
	_ "github.com/syntaxgame/dragon-legend/factory"
	"github.com/syntaxgame/dragon-legend/logging"
	"github.com/syntaxgame/dragon-legend/nats"
	"github.com/syntaxgame/dragon-legend/redis"
	"github.com/thoas/go-funk"
)

var (
	logger = logging.Logger
)

func initDatabase() {
	for {
		err := database.InitDB()
		if err == nil {
			log.Printf("Connected to database...")
			return
		}
		log.Printf("Database connection error: %+v, waiting 30 sec...", err)
		time.Sleep(time.Duration(30) * time.Second)
	}
}

func initRedis() {
	for {
		err := redis.InitRedis()
		if err != nil {
			log.Printf("Redis connection error: %+v, waiting 30 sec...", err)
			time.Sleep(time.Duration(30) * time.Second)
			continue
		}

		if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
			log.Printf("Connected to redis...")
			go logger.StartLogging()
		}

		return
	}
}

func startServer() {
	cfg := config.Default
	port := cfg.Server.Port

	listen, err := net.Listen("tcp4", ":"+strconv.Itoa(port))
	defer listen.Close()
	if err != nil {
		log.Fatalf("Socket listen port %d failed,%s", port, err)
		os.Exit(1)
	}
	log.Printf("Begin listen port: %d", port)

	//connections = make(map[string]net.Conn)
	//remoteAddrs = make(map[string]int)

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatalln(err)
			continue
		}

		ws := database.Socket{Conn: conn}
		//ws.SetPingDuration(time.Second * 2)
		//ws.SetPingHandler(nil)
		go ws.Read()
	}
}

func cronHandler() {
	c := cron.New()
	c.AddFunc("0 0 0 * * *", func() {
		database.RefreshAIDs()
	})
	c.Start()
}

func main() {

	initRedis()
	initDatabase()
	cronHandler()

	ai.Init()
	go database.UnbanUsers()

	s := nats.RunServer(nil)
	defer s.Shutdown()

	c, err := nats.ConnectSelf(nil)
	defer c.Close()

	if err != nil {
		log.Fatalln(err)
	}

	go api.InitGRPC()

	startServer()
}

func resolveOverlappingItems() { //67-306
	ids := []string{}

	for _, userid := range ids {
		fmt.Println("user id:", userid)
		bankSlots, _ := database.FindBankSlotsByUserID(userid)
		freeSlots := make(map[int16]struct{})
		for _, s := range bankSlots {
			freeSlots[s.SlotID] = struct{}{}
		}

		findSlot := func() int16 {
			for i := int16(67); i <= 306; i++ {
				if _, ok := freeSlots[i]; !ok {
					return i
				}
			}
			return -1
		}

		for i := 0; i < len(bankSlots)-1; i++ {
			for j := i; true; j++ {
				if len(bankSlots) == j+1 || bankSlots[i].SlotID != bankSlots[j+1].SlotID {
					break
				}

				free := findSlot()
				if free == -1 {
					continue
				}

				fmt.Printf("%d => %d\n", bankSlots[j+1].SlotID, free)
				freeSlots[free] = struct{}{}
				bankSlots[j+1].SlotID = free
				bankSlots[j+1].Update()
			}
		}
	}
}

func createServerMobs(server int) {
	aiSet := funk.Filter(funk.Values(database.AIs), func(ai *database.AI) bool {
		return ai.Server == 1
	}).([]*database.AI)

	sort.Slice(aiSet, func(i, j int) bool {
		return aiSet[i].ID < aiSet[j].ID
	})

	for _, ai := range aiSet {
		newAI := *ai
		newAI.ID = 0
		newAI.Server = server
		err := newAI.Create()
		if err != nil {
			log.Print(err)
		}
	}
}
