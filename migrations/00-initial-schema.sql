-- liquibase formatted sql
-- changeset z0tedd:00-initial-schema

-- up:
CREATE TABLE IF NOT EXISTS users_preferences (
    user_id BIGINT NOT NULL,
    sub_id BIGINT NOT NULL,
    filters TEXT[] NOT NULL,
    tags TEXT[] NOT NULL,
    url TEXT NOT NULL,
    PRIMARY KEY (user_id, sub_id)
);

CREATE TABLE IF NOT EXISTS subscriptions (
    sub_id BIGINT PRIMARY KEY,
    url TEXT NOT NULL,
    tg_chat_ids BIGINT[] NOT NULL,
    last_activity JSONB NOT NULL
);

-- down:
DROP TABLE IF EXISTS users_preferences;
DROP TABLE IF EXISTS subscriptions;
