-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    bio TEXT,
    bio_link VARCHAR(255),
    username VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100),
    profile_image_url TEXT,
    life_aspirations TEXT,
    things_i_like_to_do TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_logged_in BOOLEAN DEFAULT FALSE,
    is_banned BOOLEAN DEFAULT FALSE
);

CREATE TABLE administrators (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(255) UNIQUE NOT NULL
);

CREATE TABLE aspiration_updates (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE likes (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    update_id INTEGER REFERENCES aspiration_updates(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, update_id)
);

CREATE TABLE followers (
    follower_id INTEGER REFERENCES users(id),
    followed_id INTEGER REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (follower_id, followed_id)
);

CREATE TABLE comments (
    id SERIAL PRIMARY KEY,
    update_id INTEGER REFERENCES aspiration_updates(id),
    user_id INTEGER REFERENCES users(id),
    parent_id INTEGER REFERENCES comments(id),
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE comments;
DROP TABLE followers;
DROP TABLE likes;
DROP TABLE aspiration_updates;
DROP TABLE administrators;
DROP TABLE users;
-- +goose StatementEnd
