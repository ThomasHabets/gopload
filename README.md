# gopload

This is not an official Google product.

## Build

```
$ go get github.com/Sirupsen/logrus
$ go get github.com/gorilla/websocket
$ go get github.com/gorilla/mux
$ go build *.go
```

## Install with nginx

Start gopload, listening to port 8081 (default).

Configure nginx:

```
location ^~ /upload {
    proxy_pass  http://127.0.0.1:8081;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $connection_upgrade;
}
```
