package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/OleksandrKirovychDev/gator/internal/database"
	"github.com/OleksandrKirovychDev/gator/internal/rss"
	"github.com/google/uuid"
	"github.com/lib/pq"
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

func scrapeFeeds(s *State) {
	ctx := context.Background()

	feed, err := s.dbQueries.GetNextFeedToFetch(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Println("no feeds to fetch")
			return
		}
		log.Printf("failed to get next feed to fetch: %v", err)
		return
	}

	if err := s.dbQueries.MarkFeedFetched(ctx, feed.ID); err != nil {
		log.Printf("failed to mark feed '%s' as fetched: %v", feed.Name, err)
		return
	}

	rssFeed, err := rss.FetchFeed(ctx, feed.Url)
	if err != nil {
		log.Printf("failed to fetch feed '%s' (%s): %v", feed.Name, feed.Url, err)
		return
	}

	saved := 0
	for _, item := range rssFeed.Channel.Item {
		err := s.dbQueries.CreatePost(ctx, database.CreatePostParams{
			ID:          uuid.New(),
			Title:       html.UnescapeString(item.Title),
			Url:         item.Link,
			Description: toNullString(html.UnescapeString(item.Description)),
			PublishedAt: parsePublishedAt(item.PubDate),
			FeedID:      feed.ID,
		})
		if err != nil {
			if isUniqueViolation(err) {
				continue
			}
			log.Printf("failed to save post '%s': %v", item.Link, err)
			continue
		}
		saved++
	}

	log.Printf("feed '%s' collected: %d items, %d new post(s) saved", feed.Name, len(rssFeed.Channel.Item), saved)
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func parsePublishedAt(raw string) sql.NullTime {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return sql.NullTime{}
	}

	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.RFC3339,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return sql.NullTime{Time: t, Valid: true}
		}
	}

	log.Printf("could not parse published date %q", raw)
	return sql.NullTime{}
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
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

func handlerBrowse(s *State, cmd Command, user database.User) error {
	limit := 2
	if len(cmd.args) > 0 {
		n, err := strconv.Atoi(cmd.args[0])
		if err != nil {
			return fmt.Errorf("invalid limit '%s': must be a number", cmd.args[0])
		}
		if n < 1 {
			return fmt.Errorf("limit must be a positive number")
		}
		limit = n
	}

	posts, err := s.dbQueries.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		return fmt.Errorf("failed to get posts: %w", err)
	}

	if len(posts) == 0 {
		fmt.Println("No posts found. Follow some feeds and run `agg` to collect posts.")
		return nil
	}

	fmt.Printf("Found %d post(s) for user %s:\n\n", len(posts), user.Name)
	for _, post := range posts {
		published := "unknown date"
		if post.PublishedAt.Valid {
			published = post.PublishedAt.Time.Format("Mon Jan 2, 2006")
		}

		fmt.Printf("%s · %s\n", published, post.FeedName)
		fmt.Printf("  %s\n", post.Title)
		fmt.Printf("  %s\n", post.Url)
		if post.Description.Valid && strings.TrimSpace(post.Description.String) != "" {
			fmt.Printf("  %s\n", post.Description.String)
		}
		fmt.Println("=====================================")
	}

	return nil
}

func followFeed(s *State, userID, feedID uuid.UUID) (database.CreateFeedFollowRow, error) {
	return s.dbQueries.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:     uuid.New(),
		UserID: userID,
		FeedID: feedID,
	})
}
