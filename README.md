# LanOps OBS Stream Proxy

RTMP server & bridge to automatically (if enabled) inject RTMP streams into OBS under a scene as a media source. When rotation is active, it will rotate visibility of each stream.

Intended for use at [LanOps Events](https://www.lanops.co.uk)

Thanks to everyone involved in the following projects:

https://github.com/bluenviron/mediamtx
https://github.com/andreykaipov/goobs

## Prerequisites

- Mediamtx must be running with the API enabled (docker compose available in ```resources/docker/compose```)
- OBS must be running
- ```cp src/.env.example src/.env``` and fill it in

### Install Dependencies

```bash
cd src
go mod tidy
```

## Usage

Entry Point:
```bash
go run ./cmd/stream-proxy
```

## API Endpoints

| Endpoint                | Method | URL Params | JSON Input            | Description                                |
|-------------------------|--------|------------|-----------------------|--------------------------------------------|
| `/streams`              | GET    | None       | None                  | Retrieves a list of all available streams. |
| `/streams/:name`        | GET    | `name`     | None                  | Retrieves details for a specific stream.   |
| `/streams/:name/enable` | POST   | `name`     | `{ "enabled": bool }` | Enables/Disables the specified stream.     |

## Env

| Variable                 | Description                               |
|--------------------------|-------------------------------------------|
| `DB_PATH`                | Path to the SQLite database file.         |
| `OBS_WEBSOCKET_ADDRESS`  | Address of the OBS WebSocket server.      |
| `OBS_WEBSOCKET_PASSWORD` | Password for authenticating with OBS.     |
| `OBS_PROXY_SCENE_NAME`   | Name of the scene used as a proxy in OBS. |
| `MEDIAMTX_API_ADDRESS`   | Address of the MediaMTX API endpoint.     |
| `MEDIAMTX_RTMP_ADDRESS`  | RTMP address for MediaMTX streaming.      |

## Docker

```docker build -f resources/docker/Dockerfile .```

```
docker run -d \
  --name jukebox-service \
  --restart unless-stopped \
  -e DB_PATH=bridge.db \
  -e OBS_WEBSOCKET_ADDRESS= \
  -e OBS_WEBSOCKET_PASSWORD= \
  -e OBS_PROXY_SCENE_NAME="Proxy Scenes" \
  -e MEDIAMTX_API_ADDRESS= \
  -e MEDIAMTX_RTMP_ADDRESS= \
  -p 8080:8080 \
  th0rn0/lanops-spotify-jukebox:service-latest
```

```
  stream-proxy:
    image: th0rn0/lanops-stream-proxy:latest
    container_name: stream-proxy
    restart: unless-stopped
    environment:
      DB_PATH: "bridge.db"
      OBS_WEBSOCKET_ADDRESS: 
      OBS_WEBSOCKET_PASSWORD: 
      OBS_PROXY_SCENE_NAME: "Proxy Scenes"
      MEDIAMTX_API_ADDRESS: 
      MEDIAMTX_RTMP_ADDRESS: 
    ports:
      - 8080:8080
```

## RTMP Stream Generator

In the ```resources/tools``` dir there is a go script to generate RTMP streams. 

```
go run resources/tools/rtmpStreamGenerator.go
```
