-- +goose Up
ALTER TABLE users
    ADD CONSTRAINT users_unique_name UNIQUE (name);

-- +goose Down
ALTER TABLE users
    DROP CONSTRAINT users_unique_name;