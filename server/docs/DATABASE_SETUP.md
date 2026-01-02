# Database Setup Guide

This server supports both SQLite (for local development) and PostgreSQL (for production/cloud deployment).

## Local Development (SQLite)

By default, the server uses SQLite with a local file. No configuration needed:

```bash
./simulation_server
```

This will create `scenarios.db` in the current directory.

## Production/Cloud Deployment (PostgreSQL)

For production deployments on AWS ECS or other cloud platforms, use PostgreSQL.

### Option 1: Amazon RDS PostgreSQL (Recommended)

Amazon RDS is the most common and recommended database solution for ECS deployments.

#### 1. Create RDS PostgreSQL Instance

Using AWS Console:
1. Go to RDS in AWS Console
2. Click "Create database"
3. Choose "PostgreSQL" as engine
4. Select a template (e.g., "Free tier" for development)
5. Configure:
   - DB instance identifier: `simulation-db`
   - Master username: `postgres` (or your choice)
   - Master password: (set a strong password)
   - DB instance class: `db.t3.micro` (or larger for production)
   - Storage: 20 GB (adjust as needed)
   - VPC: Use the same VPC as your ECS cluster
   - Public access: Set based on your security requirements
   - Security group: Create/select one that allows PostgreSQL (port 5432) from your ECS tasks

#### 2. Get Connection String

After creation, get the endpoint from RDS console. The connection string format is:

```
postgres://username:password@hostname:5432/dbname?sslmode=require
```

Example:
```
postgres://postgres:mypassword@simulation-db.abc123.us-east-1.rds.amazonaws.com:5432/postgres?sslmode=require
```

#### 3. Configure ECS Task Definition

In your ECS task definition, set the `DATABASE_URL` environment variable:

```json
{
  "environment": [
    {
      "name": "DATABASE_URL",
      "value": "postgres://postgres:password@hostname:5432/dbname?sslmode=require"
    }
  ]
}
```

**Security Best Practice**: Use AWS Secrets Manager or Parameter Store instead of hardcoding credentials:

```json
{
  "secrets": [
    {
      "name": "DATABASE_URL",
      "valueFrom": "arn:aws:secretsmanager:region:account:secret:db-connection-string"
    }
  ]
}
```

### Option 2: Amazon Aurora Serverless PostgreSQL

For auto-scaling workloads, consider Aurora Serverless:

1. Create Aurora Serverless PostgreSQL cluster
2. Use the same connection string format
3. Aurora automatically scales based on demand

### Option 3: Self-Managed PostgreSQL in ECS

You can also run PostgreSQL in a separate ECS task, but RDS is recommended for production.

## Connection String Formats

### PostgreSQL
```
postgres://username:password@hostname:port/database?sslmode=require
```

Common SSL modes:
- `sslmode=require` - Require SSL (recommended for RDS)
- `sslmode=disable` - No SSL (only for local/testing)
- `sslmode=prefer` - Prefer SSL but allow non-SSL

### SQLite
```
scenarios.db
```
or
```
/path/to/scenarios.db
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | Database connection string | `scenarios.db` (SQLite) |
| `PORT` | Server port | `3000` |

## Database Schema

The server automatically creates the required table on startup:

```sql
CREATE TABLE scenarios (
    id SERIAL PRIMARY KEY,              -- PostgreSQL
    -- id INTEGER PRIMARY KEY AUTOINCREMENT,  -- SQLite
    name TEXT NOT NULL,
    yaml_content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP  -- PostgreSQL
    -- created_at TEXT DEFAULT (datetime('now'))  -- SQLite
);
```

## Migration from SQLite to PostgreSQL

If you have existing data in SQLite:

1. Export data from SQLite:
   ```bash
   sqlite3 scenarios.db .dump > dump.sql
   ```

2. Convert SQL syntax (if needed) and import to PostgreSQL:
   ```bash
   psql -h hostname -U username -d dbname < dump.sql
   ```

## Testing Database Connection

Test your PostgreSQL connection:

```bash
# Using psql
psql "postgres://username:password@hostname:5432/dbname?sslmode=require"

# Or set environment variable and run server
export DATABASE_URL="postgres://username:password@hostname:5432/dbname?sslmode=require"
./simulation_server
```

## Troubleshooting

### Connection Refused
- Check security group allows PostgreSQL port (5432) from ECS tasks
- Verify RDS instance is in the same VPC or has proper network configuration
- Check RDS instance is publicly accessible (if needed) or use VPC peering

### Authentication Failed
- Verify username and password are correct
- Check RDS master username matches connection string
- Ensure database name exists

### SSL Required
- RDS requires SSL by default - use `sslmode=require` in connection string
- For local testing, you can use `sslmode=disable` (not recommended for production)

## Security Recommendations

1. **Use AWS Secrets Manager** for database credentials in ECS
2. **Enable SSL** for all production connections (`sslmode=require`)
3. **Use VPC Security Groups** to restrict database access to ECS tasks only
4. **Enable RDS encryption at rest**
5. **Regular backups** - RDS provides automated backups
6. **Use IAM database authentication** if possible (advanced)
