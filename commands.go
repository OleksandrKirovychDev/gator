package main

import (
	"fmt"

	"github.com/OleksandrKirovychDev/gator/internal/database"
)

type Command struct {
	name string
	args []string
}

type commandHandler func(s *State, cmd Command) error
type authedHandler func(s *State, cmd Command, user database.User) error

type Commands struct {
	commands map[string]commandHandler
}

func (c *Commands) Run(s *State, cmd Command) error {
	handler, exists := c.commands[cmd.name]
	if !exists {
		return fmt.Errorf("unknown command: %s", cmd.name)
	}
	return handler(s, cmd)
}

func (c *Commands) Register(name string, handler commandHandler) {
	if c.commands == nil {
		c.commands = make(map[string]commandHandler)
	}
	c.commands[name] = handler
}
