# fem-fd-service

Example Go application from Fullstack Deployment: From Containers to Production AWS on Frontend Masters.

## goals

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

### Docker Setup

To run local services (postgres, etc) and migrations:

```bash
make down # clears environment
make up # starts services
make migrate # runs SQL script
```

Then once you login to the app, add yourself as admin:

```sql
INSERT INTO administrators (email, username)
SELECT email, username
FROM users
WHERE email = 'user@example.com';
```

### Development Setup

1. Copy `.env.example` to create new `.env` file
2. Update `.env` file with OAuth credentials
3. Source `.env` with `source .env`
4. Start server with `make start`
5. Navigate to http://localhost:8080

## License

This project is proprietary and closed source. All rights reserved. Unauthorized use, reproduction, or distribution of this software is strictly prohibited.
