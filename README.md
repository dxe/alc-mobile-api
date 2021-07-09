# ALC Mobile API
Backend service for the ALC app

# Local development
1. Copy docker-compose.dev-example.yml to docker-compose.yml.
2. Fill in the OAUTH_CLIENT_ID and OAUTH_CLIENT_SECRET environment variables.
3. Run ``make dev`` to build the image and start the server. (Note: You may receive an error the first time you start the server due to MySQL not being fully up in time. If this happens, simply wait for the MySQL container to be fully up, stop the containers, then run ``make dev`` again.)
4. When you modify .go files, the server automatically rebuilds and restarts.

# Wiping local database
1. Ensure the Docker containers are stopped.
2. Run ``make rm_db``.