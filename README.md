# ALC Mobile API
Backend service for the ALC app

# Local development
1. Copy docker-compose.dev-example.yml to docker-compose.yml.
2. Fill in the OAUTH_CLIENT_ID and OAUTH_CLIENT_SECRET environment variables.
3. Run ``docker-compose up --build`` to build the image and start the server.
4. When you modify .go files, the server automatically rebuilds and restarts.
