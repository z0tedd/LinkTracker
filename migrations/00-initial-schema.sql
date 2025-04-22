-- Создание таблицы users_preferences
CREATE TABLE users_preferences (
    userID BIGINT NOT NULL,
    subID BIGINT NOT NULL,
    filters TEXT[] NOT NULL,
    tags TEXT[] NOT NULL,
    url TEXT NOT NULL,
    PRIMARY KEY (userID, subID) -- Составной первичный ключ для уникальности пары userID и subID
);

-- Создание таблицы subscriptions
CREATE TABLE subscriptions (
    subID BIGINT PRIMARY KEY,
    url TEXT NOT NULL,
    tgChatIDs BIGINT[] NOT NULL,
    lastActivity JSONB NOT NULL
);
