package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func middlewareLoggedIn(handler authedHandler) commandHandler {
	return func(s *State, cmd Command) error {
		currentUser := s.config.CurrentUser
		if currentUser == "" {
			return fmt.Errorf("no user is currently logged in")
		}
		user, err := s.dbQueries.GetUser(context.Background(), currentUser)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("current user '%s' does not exist", currentUser)
			}
			return fmt.Errorf("failed to get current user: %w", err)
		}
		return handler(s, cmd, user)
	}
}

func requireArgs(cmd Command, usage string, n int) error {
	if len(cmd.args) < n {
		return fmt.Errorf("usage: %s %s", cmd.name, usage)
	}
	return nil
}
