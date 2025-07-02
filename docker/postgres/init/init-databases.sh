#!/bin/bash
set -e # Stop the script on any error

# create_database - function to create a database if it does not exist
create_database() {
    local database=$1
    echo "  Creating database '$database'"
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
        SELECT 'CREATE DATABASE $database'
        WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$database')\gexec
EOSQL
}

# list of databases to create
databases=$(echo "products_db,orders_db" | tr ',' ' ')

if [ -n "$databases" ]; then
    echo "Multiple database creation requested: $databases"
    for db in $databases; do
        create_database "$db"
    done
    echo "Multiple databases created"
fi
