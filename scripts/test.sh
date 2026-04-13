#!/bin/sh

PROJECT="taskflow-test"

echo "Cleaning up previous test containers..."
docker compose -p "$PROJECT" -f docker-compose.test.yml down -v 2>/dev/null

echo "Starting integration tests..."
docker compose -p "$PROJECT" -f docker-compose.test.yml up --build --exit-code-from test test
EXIT_CODE=$?

echo "Cleaning up..."
docker compose -p "$PROJECT" -f docker-compose.test.yml down -v

exit $EXIT_CODE
