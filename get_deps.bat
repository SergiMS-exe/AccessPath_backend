@echo off
cd /d d:\Git\AccessPath_backend
go get github.com/redis/go-redis/v9@latest
go get github.com/chai2010/webp@latest
go mod tidy
