[![Frontend Masters](https://static.frontendmasters.com/assets/brand/logos/full.png)](https://frontendmasters.com/courses/fullstack-deployment)
This is a companion repository for the [Modern Deployment](https://frontendmasters.com/courses/fullstack-deployment) course on Frontend Masters. The repository contains an example Go application that will be deployed to AWS.

## goals Application

This application is a social media platform for setting and sharing life goals and aspirations.

### Features

- User authentication with Google OAuth2
- Create and edit personal profiles (username, display name, bio, bio link, life aspirations, things I like to do)
- Share aspiration updates (create, edit, and delete)
- Leave nested comments on aspiration updates
- Like and unlike updates
- Follow and unfollow other users
- Browse recent users and updates
- User banning system (admin functionality)

### Prerequisites for Running the Application

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


## Course Resources

The resources and code snippets below are reference throughout the [Fullstack Deployment: From Containers to Production AWS]() course.

### General Setup Instructions

To follow along with the course, you will need:

1. Go version 1.24.2 or later
    - [Download and install Go](https://go.dev/doc/install)
1. Docker Desktop
    - [Download and install Docker Desktop](https://www.docker.com/products/docker-desktop/)
    - **Important:** Check "Enable host networking" under `Settings > Resources > Network`
1. Google Cloud Console
    - Log into the [Google Cloud Console](https://console.cloud.google.com/auth/clients)
    - You'll create an OAuth Client during the course
1. AWS
    - Create an AWS Root User account and log into the [AWS Console](https://us-east-1.console.aws.amazon.com/console/home)
    - Install the [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
    - In the course, you'll create an Administrator User in IAM for the CLI and [set environment variables](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html) in your console to authenticate the CLI
1. Terraform
    - Install the [Terraform CLI](https://developer.hashicorp.com/terraform/install)
1. Supabase
    - Create a [Supabase account]()
1.


### App Runner IAM Policy

In the **AWS Parameter Store** lesson, copy and paste this JSON code when you are creating the IAM policy for the AWS App Runner:

```json
{
	"Statement": [
		{
			"Action": ["ssm:GetParameters"],
			"Effect": "Allow",
			"Resource": ["arn:aws:ssm:us-west-2:<ACCOUNT_ID>:parameter/fem-fd-service/*"]
		}
	],
    "Version": "2012-10-17"
}
```

### App Runner IAM Role

In the **App Runner IAM Role** lesson, copy and paste this JSON code when you are creating the IAM Role for the AWS App Runner:

```json
{
    "Version": "2012-10-17",
	"Statement": [
		{
			"Effect": "Allow",
            "Principal": {
                "Service": ["tasks.apprunner.amazonaws.com"]
            },
			"Action": ["sts:AssumeRole"],
		}
	],
}
```

### Adding Database Migration

The application uses [goose for database migrations](https://github.com/pressly/goose). Install goose:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

Then verify the installation by running `goose -version`

**Note:** If you see a `command not found: goose` error when trying to run goose, it's because the `$HOME/go/bin` directory is not added to your PATH. You can fix this temporarily by running export `PATH=$HOME/go/bin:$PATH`, but this will not persist if you close your terminal. A permanent fix would require adding export `PATH=$HOME/go/bin:$PATH` to your .zshrc or .bashrc.


### Deploying the Service

When you are deploying the fm-fd-service with Terraform, Erik covers some troubleshooting tips throughout the lesson. Here are some additional troubleshooting tips:

**Clean up your local docker images and push up a fresh image to ECR**
```bash
# You first may need to log (Go to ECR > Click on your image > View Push Command)
# aws ecr get-login-password......

# Clean up your existing image builds
`docker system prune --all && docker buildx prune --all`

# Build a new image for staging and push it to ECR
make build-image
make BUILD_TAG="staging" build-image-promote
```

**Destroy and Reapply the Terraform configuration**
If you visit ECS and your staging cluster has 0 Container Instances (or there are any other issues you can't resolve), you likely have an issue with your Terraform configuration. Firstly, tear down the existing infrastructure:

```bash
terraform destroy
```

Confirm your terraform configuration matches [this commit on Erik's workshop branch](https://github.com/ALT-F4-LLC/fem-fd-service/tree/94ee588f97cc00f5a13aec486df08a5bc04deb22). Then redeploy the infrastructure with the init, plan and apply commands:

```bash
terraform init
terraform plan -out "terraform.tfplan"   
terraform apply "terraform.tfplan"   
```

## Deleting AWS Resources

Once you complete the course, you'll need to remove all the AWS resources to avoid changes:

1. Run `terraform destroy` to remove all resources created by Terraform
1. Navigate to [AWS App Runner](https://us-west-2.console.aws.amazon.com/apprunner/home?region=us-west-2), click on your App Runner instance and choose `Actions > Delete`
1. Navigate to [AWS Parameter Store](https://us-west-2.console.aws.amazon.com/systems-manager/parameters/?region=us-west-2) and delete the `fm-fd-service` parameters (the others should have been removed by running `terraform destroy`)
1. Navigate [to ECR](https://us-west-2.console.aws.amazon.com/ecr/private-registry/repositories?region=us-west-2) and delete your container
1. Delete your [Supabase database](https://supabase.com/dashboard/)

> You can monitor your AWS changes in the [Cost Explorer](https://us-east-1.console.aws.amazon.com/costmanagement/home?region=us-west-2#/home)

## License

This project is proprietary and closed source. All rights reserved. Unauthorized use, reproduction, or distribution of this software is strictly prohibited.
