package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"log"
	"time"

	"github.com/OleksandrKirovychDev/gator/internal/database"
	"github.com/OleksandrKirovychDev/gator/internal/rss"
	"github.com/google/uuid"
)

func handlerLogin(s *State, cmd Command) error {
	if err := requireArgs(cmd, "<username>", 1); err != nil {
		return err
	}
	username := cmd.args[0]

	_, err := s.dbQueries.GetUser(context.Background(), username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("user '%s' does not exist", username)
		}
		return fmt.Errorf("failed to check current user: %w", err)
	}

	err = s.config.SetUser(username)
	if err != nil {
		return fmt.Errorf("failed to set user: %w", err)
	}

	fmt.Printf("User '%s' logged in successfully\n", username)

	return nil
}

func handlerRegister(s *State, cmd Command) error {
	if err := requireArgs(cmd, "<username>", 1); err != nil {
		return err
	}

	username := cmd.args[0]
	user, err := s.dbQueries.CreateUser(context.Background(), database.CreateUserParams{
		ID:   uuid.New(),
		Name: username,
	})
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	err = s.config.SetUser(username)
	if err != nil {
		return fmt.Errorf("failed to set user: %w", err)
	}

	fmt.Printf("User '%s' registered successfully with ID %s\n", user.Name, user.ID)
	return nil
}

func handlerReset(s *State, cmd Command) error {
	err := s.dbQueries.ClearUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed to clear users: %w", err)
	}

	err = s.config.SetUser("")
	if err != nil {
		return fmt.Errorf("failed to reset user in config: %w", err)
	}

	fmt.Println("All users have been cleared and current user reset.")
	return nil
}

func handlerListUsers(s *State, cmd Command) error {
	users, err := s.dbQueries.GetAllUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}
	currentUser := s.config.CurrentUser

	for _, user := range users {
		if user.Name == currentUser {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}

	return nil
}

func handlerAggregate(s *State, cmd Command) error {
	if err := requireArgs(cmd, "<time_between_reqs>", 1); err != nil {
		return err
	}

	timeBetweenRequests, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return fmt.Errorf("invalid duration '%s': %w", cmd.args[0], err)
	}

	fmt.Printf("Collecting feeds every %s\n", timeBetweenRequests)

	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
}

// scrapeFeeds fetches the single feed that is most overdue for a refresh,
// marks it as fetched, and prints the titles of its posts to the console.
func scrapeFeeds(s *State) {
	feed, err := s.dbQueries.GetNextFeedToFetch(context.Background())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Println("no feeds to fetch")
			return
		}
		log.Printf("failed to get next feed to fetch: %v", err)
		return
	}

	if err := s.dbQueries.MarkFeedFetched(context.Background(), feed.ID); err != nil {
		log.Printf("failed to mark feed '%s' as fetched: %v", feed.Name, err)
		return
	}

	rssFeed, err := rss.FetchFeed(context.Background(), feed.Url)
	if err != nil {
		log.Printf("failed to fetch feed '%s' (%s): %v", feed.Name, feed.Url, err)
		return
	}

	fmt.Printf("Collecting feed '%s' (%s)\n", feed.Name, feed.Url)
	for _, item := range rssFeed.Channel.Item {
		fmt.Printf(" - %s\n", html.UnescapeString(item.Title))
	}
	fmt.Printf("Feed '%s' collected, %d posts found\n", feed.Name, len(rssFeed.Channel.Item))
}

func handlerAddFeed(s *State, cmd Command, user database.User) error {
	if err := requireArgs(cmd, "<name> <url>", 2); err != nil {
		return err
	}
	feedName := cmd.args[0]
	feedURL := cmd.args[1]

	feed, err := s.dbQueries.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:     uuid.New(),
		Name:   feedName,
		Url:    feedURL,
		UserID: user.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to create feed: %w", err)
	}

	_, err = followFeed(s, user.ID, feed.ID)
	if err != nil {
		return fmt.Errorf("failed to create feed follow: %w", err)
	}

	fmt.Printf("Feed '%s' added successfully with ID %s\n", feed.Name, feed.ID)
	return nil
}

func handlerListFeeds(s *State, cmd Command) error {
	feeds, err := s.dbQueries.GetAllFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list feeds: %w", err)
	}

	for _, feed := range feeds {
		fmt.Printf("* %s (URL: %s, User: %s)\n", feed.Name, feed.Url, feed.UserName)
	}

	return nil
}

func handlerFollowFeed(s *State, cmd Command, user database.User) error {
	if err := requireArgs(cmd, "<url>", 1); err != nil {
		return err
	}

	feedURL := cmd.args[0]

	feed, err := s.dbQueries.GetFeedByURL(context.Background(), feedURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("feed with URL '%s' does not exist", feedURL)
		}
		return fmt.Errorf("failed to get feed by URL: %w", err)
	}

	feedFollow, err := followFeed(s, user.ID, feed.ID)
	if err != nil {
		return fmt.Errorf("failed to create feed follow: %w", err)
	}

	fmt.Printf("Successfully followed feed '%s' with URL %s\n", feed.Name, feedFollow.Url)
	return nil
}

func handlerGetAllFollowingFeeds(s *State, cmd Command, user database.User) error {
	followingFeeds, err := s.dbQueries.GetFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("failed to get following feeds: %w", err)
	}

	for _, feed := range followingFeeds {
		fmt.Printf("* %s (URL: %s)\n", feed.Name, feed.Url)
	}

	return nil
}

func handlerUnfollowFeed(s *State, cmd Command, user database.User) error {
	if err := requireArgs(cmd, "<url>", 1); err != nil {
		return err
	}

	feedURL := cmd.args[0]

	feed, err := s.dbQueries.GetFeedByURL(context.Background(), feedURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("feed with URL '%s' does not exist", feedURL)
		}
		return fmt.Errorf("failed to get feed by URL: %w", err)
	}

	err = s.dbQueries.UnfollowFeed(context.Background(), database.UnfollowFeedParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to unfollow feed: %w", err)
	}

	fmt.Printf("Successfully unfollowed feed '%s' with URL %s\n", feed.Name, feed.Url)
	return nil
}

func followFeed(s *State, userID, feedID uuid.UUID) (database.CreateFeedFollowRow, error) {
	return s.dbQueries.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:     uuid.New(),
		UserID: userID,
		FeedID: feedID,
	})
}
