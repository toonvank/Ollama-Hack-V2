# Database Migration Guide: MySQL to PostgreSQL

## Overview

Ollama-Hack V2 has been updated to use **PostgreSQL** as the default database engine instead of MySQL. PostgreSQL offers better performance, improved concurrency handling, and more advanced features.

## What Changed

### Default Database Engine
- **Before:** MySQL 8.0
- **After:** PostgreSQL 16 (Alpine)

### Connection Details
- **Port:** Changed from 3306 (MySQL) to 5432 (PostgreSQL)
- **Driver:** Changed from `aiomysql` to `asyncpg`
- **Connection String:** Automatically generated based on `DATABASE__ENGINE` setting

## For New Installations

Simply follow the installation instructions in the README. PostgreSQL is now the default.

```bash
curl -o docker-compose.yml https://raw.githubusercontent.com/timlzh/ollama-hack/main/docker-compose.example.yml
docker compose up -d
```

## For Existing Installations (MySQL)

### Option 1: Start Fresh with PostgreSQL (Recommended for Testing)

1. **Backup your data** (if needed)
2. Stop the current services:
   ```bash
   docker compose down
   ```
3. Update your `docker-compose.yml`:
   - Replace `mysql:8.0` with `postgres:16-alpine`
   - Update environment variables (see example below)
4. Remove old database volume:
   ```bash
   rm -rf ./data/mysql
   ```
5. Start with PostgreSQL:
   ```bash
   docker compose up -d
   ```

### Option 2: Migrate Data from MySQL to PostgreSQL

If you have existing data you want to keep:

1. **Export MySQL data:**
   ```bash
   docker exec ollama-hack-db mysqldump -u ollama_hack -p ollama_hack > backup.sql
   ```

2. **Convert MySQL dump to PostgreSQL format:**
   You can use tools like:
   - [mysql2postgres](https://github.com/AnatolyUss/nmig)
   - [pgloader](https://pgloader.io/)
   
   Example with pgloader:
   ```bash
   pgloader mysql://ollama_hack:password@localhost/ollama_hack \
             postgresql://ollama_hack:password@localhost/ollama_hack
   ```

3. **Update docker-compose.yml** to use PostgreSQL

4. **Restart services:**
   ```bash
   docker compose down
   docker compose up -d
   ```

### Option 3: Continue Using MySQL

PostgreSQL is now the **default**, but MySQL is still supported:

Update your environment variables in `docker-compose.yml`:

```yaml
environment:
  - DATABASE__ENGINE=mysql
  - DATABASE__HOST=db
  - DATABASE__PORT=3306
  - DATABASE__USERNAME=ollama_hack
  - DATABASE__PASSWORD=change_this_password
  - DATABASE__DB=ollama_hack
```

And keep the MySQL service configuration:

```yaml
db:
  image: mysql:8.0
  environment:
    - MYSQL_ROOT_PASSWORD=root_password
    - MYSQL_DATABASE=ollama_hack
    - MYSQL_USER=ollama_hack
    - MYSQL_PASSWORD=change_this_password
  volumes:
    - ./data/mysql:/var/lib/mysql
  command: --default-authentication-plugin=mysql_native_password
```

## Updated Docker Compose Configuration

### PostgreSQL (Default)

```yaml
services:
  backend:
    environment:
      - DATABASE__ENGINE=postgresql
      - DATABASE__HOST=db
      - DATABASE__PORT=5432
      - DATABASE__USERNAME=ollama_hack
      - DATABASE__PASSWORD=change_this_password
      - DATABASE__DB=ollama_hack

  db:
    image: postgres:16-alpine
    environment:
      - POSTGRES_DB=ollama_hack
      - POSTGRES_USER=ollama_hack
      - POSTGRES_PASSWORD=change_this_password
    volumes:
      - ./data/postgres:/var/lib/postgresql/data
```

## Performance Benefits

PostgreSQL offers several advantages:

- **Better Concurrency:** MVCC (Multi-Version Concurrency Control) provides better handling of concurrent requests
- **Advanced Features:** JSON support, full-text search, better indexing
- **Performance:** Generally faster for read-heavy workloads like endpoint testing and model queries
- **Reliability:** More robust transaction handling and data integrity

## Dependencies

The following changes were made to support PostgreSQL:

**Added:**
- `asyncpg` (>=0.30.0,<0.31.0) - Async PostgreSQL driver

**Removed:**
- `aiomysql` - MySQL async driver (no longer needed by default)

## Troubleshooting

### Connection Issues

If you see connection errors, verify:
1. PostgreSQL container is running: `docker ps`
2. Environment variables are correct in `docker-compose.yml`
3. Port 5432 is not already in use: `netstat -tulpn | grep 5432`

### Data Volume Issues

If PostgreSQL won't start, check the data volume permissions:
```bash
sudo chown -R 999:999 ./data/postgres
```

### Rolling Back to MySQL

If you need to roll back:
1. Stop services: `docker compose down`
2. Restore your `docker-compose.yml` from backup
3. Restore data volume from backup
4. Start services: `docker compose up -d`

## Support

For issues or questions:
- Create an issue on [GitHub](https://github.com/timlzh/ollama-hack)
- Check existing issues for similar problems

## Compatibility

Both databases are fully supported:
- ✅ **PostgreSQL** (Recommended, default)
- ✅ **MySQL** (Still supported)

The application code is database-agnostic and will work with either engine.
