# ALC Mobile API
Backend service for the ALC app

# Local development
1. Copy docker-compose.dev-example.yml to docker-compose.yml.
2. Fill in the OAUTH_CLIENT_ID and OAUTH_CLIENT_SECRET environment variables.
3. For S3 image uploads to work locally, set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY (the app uses the AWS SDK's default credential chain; in production this resolves to the ECS task role instead).
4. Ensure Docker is running, then run ``make dev`` to build the image and start the server.
5. When you modify .go files, the server automatically rebuilds and restarts.

# Wiping local database
1. Ensure the Docker containers are stopped.
2. Run ``make rm_db``.
