package config

import (
	"log"
	"os"
	"strconv"
)

var Default = &config{
	Database: Database{
		Driver:          "postgres",
		IP:              os.Getenv("POSTGRES_HOST"),
		Port:            getPort(),
		User:            os.Getenv("POSTGRES_USER"),
		Password:        os.Getenv("POSTGRES_PASSWORD"),
		Name:            os.Getenv("POSTGRES_DB"),
		ConnMaxIdle:     96,
		ConnMaxOpen:     144,
		ConnMaxLifetime: 10,
		Debug:           false,
		SSLMode:         "require",
	},
	Server: Server{
		IP:   os.Getenv("SERVER_IP"),
		Port: 4510,
	},
}

func getPort() int {
	sPort := os.Getenv("POSTGRES_PORT")
	port, err := strconv.ParseInt(sPort, 10, 32)
	if err != nil {
		log.Fatalln(err)
	}

	return int(port)
}
