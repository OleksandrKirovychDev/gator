package main

import (
	"database/sql"
	"os"

	"github.com/OleksandrKirovychDev/gator/internal/config"
	"github.com/OleksandrKirovychDev/gator/internal/database"
	_ "github.com/lib/pq"
)

type State struct {
	config 		*config.Config
	db    		*sql.DB
	dbQueries 	*database.Queries
}


func main() {
	cfg, err := config.Read()
	if err != nil {
		panic(err)
	}

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	dbQueries := database.New(db)

	state := &State{
		config: &cfg,
		db: db,
		dbQueries: dbQueries,
	}

	commands := &Commands{}
	commands.Register("login", handlerLogin)
	commands.Register("register", handlerRegister)
	commands.Register("reset", handlerReset)
	commands.Register("users", handlerListUsers)
	commands.Register("agg", handlerAggregate)
	commands.Register("addfeed", middlewareLoggedIn(handlerAddFeed))
	commands.Register("feeds", handlerListFeeds)
	commands.Register("follow", middlewareLoggedIn(handlerFollowFeed))
	commands.Register("following", middlewareLoggedIn(handlerGetAllFollowingFeeds))
	commands.Register("unfollow", middlewareLoggedIn(handlerUnfollowFeed))

	args := os.Args
	if len(args) < 2 {
		println("Error: No command provided")
		os.Exit(1)
		return
	}

	name, args := args[1], args[2:]

	cmd := Command{
		name: name,
		args: args,
	}

	err = commands.Run(state, cmd)
	if err != nil {
		println("Error:", err.Error())
		os.Exit(1)
	}
}