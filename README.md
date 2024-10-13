# Proxy HTTP Server-Client

## Run & Build
Docker run
```bash
docker run -p 8080:8080 suchimauz/proxy-http-server-client:v1.0.0
```

Local run project
```bash
make run
```

Local build project
```bash
make build
```

## Request
The service has only one endpoint
```
POST /proxify
```
Supported parameters:
```
url:           required    # url
method:        required    # request method | Supported methods: GET, POST, PUT, DELETE, PATCH
params:        optional    # query parameters (will be converted to the <url>?key=value format)
headers:       optional    # request headers (key - value), for example: {"Authorization": "Bearer <token></token>"}
body:          optional    # json request body (if necessary)
proxy:         optional    # proxy for the request
response_type: optional    # response type | supported - binary, json. Default: json
```

Example request body
```json
{
    "url": "https://api.ipify.org",
    "method": "get",
    "params": {
        "format": "json"
    },
    "headers": {
      "Accept": "application/json"
    }
}
```
Response (your **url** response must be in json format, otherwise you will get an error. The json response body will be under the key _**response**_)
```json
{
    "request": {
        "url": "https://api.ipify.org",
        "method": "get",
        "body": null,
        "headers": {
          "Accept": "application/json"
        },
        "params": {
            "format": "json"
        },
        "proxy": null
    },
    "response": {
        "ip": "<requester ip>"
    }
}
```

## Proxy
Supported parameters
```
type: required     # Supported types: http, socks5
host: required     # IP or Domain address
port: optional     # Port number | Integer

# You can also fill out authorization for your proxy
username: optional # Username for proxy authentication
password: optional # Password for proxy authentication
```

Example with proxy with auth
```json
{
    "url": "https://api.ipify.org",
    "method": "get",
    "params": {
        "format": "json"
    },
    "headers": {
      "Accept": "application/json"
    },
    "proxy": {
      "type": "socks5",
      "host": "<ip>",
      "port": 12324,
      "username": "<username>",
      "password": "<password>"
    }
}
```
Response
```json
{
    "request": {
        "url": "https://api.ipify.org",
        "method": "get",
        "body": null,
        "headers": {
          "Accept": "application/json"
        },
        "params": {
            "format": "json"
        },
        "proxy": {
          "type": "socks5",
          "host": "<ip>>",
          "port": 12324,
          "username": "<username>",
          "password": "<password>"
        }
    },
    "response": {
        "ip": "<your proxy ip>"
    }
}
```

Example proxy without auth
```
{
  "type": "socks5",
  "host": "<ip>",
  "port": 12324
}
```

Example proxy with only host (for example: http proxies)
```
{
  "type": "http",
  "host": "<ip>"
}
```

## Binary responses (images, files, etc...)

Example for request image
```json
{
    "url": "https://via.placeholder.com/150",
    "method": "get",
    "response_type": "binary",
    "proxy": {
      "type": "socks5",
      "host": "<ip>",
      "port": 12324,
      "username": "<username>",
      "password": "<password>"
    }
}
```

Response

![Response image](https://via.placeholder.com/150)