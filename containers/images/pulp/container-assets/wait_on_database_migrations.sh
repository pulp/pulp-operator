#!/bin/bash

echo "Checking for database migrations"
while true; do
  /usr/local/bin/pulpcore-manager showmigrations | grep '\[ \]'
  exit_code=$?
  if [ $exit_code -eq 1 ]; then
    # grep returning 1 means that the searched-for string was not found.
    echo "Database migrated!"
    exit 0
  elif [ $exit_code -eq 0 ]; then
    # grep returning 0 means that the searched-for string was found.
    echo "Database migration in progress. Waiting..."
  else
    # grep returning 2 or more means "error", and is probably because pulpcore-manager errored,
    # which is probably because the database is not "up enough" to continue yet.
    echo "Waiting for migration, last exit code $exit_code"
  fi
  sleep 5
done
