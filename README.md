# fem-fd-service

Example Go application from Fullstack Deployment: From Containers to Production AWS on Frontend Masters.

## goalzi

A platform for setting and sharing life goals and aspirations.

### Features

- User authentication with Google OAuth2
- Create and edit personal profiles (username, display name, bio, bio link, life aspirations, things I like to do)
- Share aspiration updates (create, edit, and delete)
- Leave nested comments on aspiration updates
- Like and unlike updates
- Follow and unfollow other users
- Browse recent users and updates
- User banning system (admin functionality)

### Prerequisites

- Go 1.24.2 or later
- Docker and Docker Compose
- PostgreSQL (if not using Docker)
- Google Cloud Console account for OAuth2 setup

### Docker Setup

1. Ensure Docker and Docker Compose are installed
2. Run `docker-compose up --detach` to start both the PostgreSQL database

### Google OAuth2 Setup

1. Go to the Google Cloud Console: https://console.cloud.google.com/
2. Create a new project or select an existing one
3. Navigate to "APIs & Services" > "Credentials"
4. Click on "Create Credentials" and select "OAuth client ID"
5. Set up the OAuth consent screen if prompted
6. Choose "Web application" as the application type
7. Set the name for your OAuth 2.0 client
8. Add http://localhost:8080/auth/google/callback to "Authorized redirect URIs"
9. Click "Create" and note down the Client ID and Client Secret
10. Keep note of credentials to use in `.env` file later

### Database Setup

```bash
docker compose exec postgres psql -U postgres -d postgres
```

```sql
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
```

Then once you login to the app, add yourself as admin

```sql
INSERT INTO administrators (email, username)
SELECT email, username
FROM users
WHERE email = 'user@example.com';
```

### Development Environment

1. Copy `.env.example` to create new `.env` file
2. Update `.env` file with OAuth credentials
3. Source `.env` with `source .env`
4. Start server with `go run main.go`
5. Navigate to http://localhost:8080

## License

This project is proprietary and closed source. All rights reserved. Unauthorized use, reproduction, or distribution of this software is strictly prohibited.
