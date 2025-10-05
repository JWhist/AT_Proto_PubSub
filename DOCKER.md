# Docker Configuration Guide

This document explains how to use the AT Protocol PubSub Server with Docker and the new dynamic configuration system.

## Quick Start

### Basic Usage
```bash
# Build the image
docker build -t at-proto-pubsub .

# Run with default Docker-optimized configuration
docker run -p 8080:8080 at-proto-pubsub
```

### Custom Configuration
```bash
# Run with custom configuration file
docker run -p 8080:8080 \
  -v /path/to/your/config.yaml:/app/config/custom.yaml:ro \
  -e CONFIG_FILE=/app/config/custom.yaml \
  at-proto-pubsub
```

## Configuration Options

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CONFIG_FILE` | `/app/config-docker.yaml` | Path to configuration file |
| `SERVER_HOST` | `0.0.0.0` | Host to bind to |
| `SERVER_PORT` | `8080` | Port to listen on |

### Built-in Configuration Files

| File | Purpose | Connection Limit | CORS | Logging |
|------|---------|------------------|------|---------|
| `config-docker.yaml` | Default Docker setup | 1000 | Allow all | JSON, structured |
| `config-default.yaml` | Original localhost config | 1000 | Allow all | Text, unstructured |

### Example Configuration Files

The `config/` directory contains example configurations:

- **`production.yaml`**: Production settings with restricted CORS and higher limits (5000 connections)
- **`high-capacity.yaml`**: High-throughput setup with 10000 connections and minimal logging
- **`load-balanced.yaml`**: Configuration optimized for load-balanced deployments (2500 connections per instance)

## Docker Compose Examples

### Single Instance
```yaml
version: '3.8'
services:
  at-proto-pubsub:
    build: .
    ports:
      - "8080:8080"
    environment:
      - CONFIG_FILE=/app/config/production.yaml
    volumes:
      - ./config/production.yaml:/app/config/production.yaml:ro
```

### Load Balanced Setup
```yaml
version: '3.8'
services:
  app1:
    build: .
    ports:
      - "8081:8080"
    environment:
      - CONFIG_FILE=/app/config/load-balanced.yaml
    volumes:
      - ./config/load-balanced.yaml:/app/config/load-balanced.yaml:ro
  
  app2:
    build: .
    ports:
      - "8082:8080"
    environment:
      - CONFIG_FILE=/app/config/load-balanced.yaml
    volumes:
      - ./config/load-balanced.yaml:/app/config/load-balanced.yaml:ro
```

### High Capacity Setup
```yaml
version: '3.8'
services:
  at-proto-pubsub:
    build: .
    ports:
      - "8080:8080"
    environment:
      - CONFIG_FILE=/app/config/high-capacity.yaml
    volumes:
      - ./config/high-capacity.yaml:/app/config/high-capacity.yaml:ro
    deploy:
      resources:
        limits:
          memory: 2G
          cpus: '1.0'
```

## Health Checks

The container includes built-in health checks:

```dockerfile
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${SERVER_PORT}/api/status || exit 1
```

## Configuration Best Practices

### Development
- Use `config-docker.yaml` or create a custom dev config
- Set `max_connections` to a lower value for testing
- Use `allow_all_origins: true` for CORS
- Set logging to `debug` or `info` level

### Production
- Use `config/production.yaml` as a starting point
- Set `max_connections` based on your expected load
- Configure specific `allowed_origins` for CORS
- Use `warn` or `error` logging level
- Use `json` format with `structured: true` for log aggregation

### High Load
- Use `config/high-capacity.yaml` as a baseline
- Set `max_connections` to 5000+ based on server capacity
- Set logging to `error` level only for performance
- Consider load balancing with multiple instances

## Monitoring

### API Endpoints
- Health: `GET /api/status`
- Statistics: `GET /api/stats`
- Active subscriptions: `GET /api/subscriptions`

### Example Stats Response
```json
{
  "success": true,
  "data": {
    "max_connections": 5000,
    "total_connections": 1250,
    "available_connections": 3750,
    "connection_utilization": "25.0%",
    "active_filters": 45
  }
}
```

## Troubleshooting

### Common Issues

1. **Connection refused**: Ensure the container is binding to `0.0.0.0`, not `localhost`
2. **CORS errors**: Update `allowed_origins` in production configurations
3. **Config not found**: Verify volume mounts and `CONFIG_FILE` environment variable
4. **Performance issues**: Check `max_connections` and consider load balancing

### Logs
```bash
# View container logs
docker logs <container_id>

# Follow logs in real-time
docker logs -f <container_id>
```