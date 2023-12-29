package main

import (
	"log"

	"github.com/pedro1287/moviemagnetbot/pkg/bot"
	"github.com/pedro1287/moviemagnetbot/pkg/db"
	"github.com/pedro1287/moviemagnetbot/pkg/http"
	"github.com/pedro1287/moviemagnetbot/pkg/model"
	"github.com/pedro1287/moviemagnetbot/pkg/movie"
	"github.com/pedro1287/moviemagnetbot/pkg/torrent"
)

func main() {

	db.Init()
	log.Printf("db inited")

	err := model.CreateSchema(db.DB)
	if err != nil {
		log.Printf("error while creating schema: %s", err)
	}

	movie.InitTMDb()
	log.Printf("tmdb inited")

	torrent.InitRARBG()
	log.Printf("rarbg inited")

	go bot.Run()
	log.Printf("bot started")

	go http.RunServer()
	log.Printf("http server started")

	select {}
}
