# Alpine
FROM golang:alpine

# copy entrypoint file
COPY entrypoint.go /usr/bin/entrypoint.go

# change mode of the entrypoint file
RUN chmod +x /usr/bin/entrypoint.go

RUN apk add zip

# set entrypoint command
ENTRYPOINT [ "go", "run", "/usr/bin/entrypoint.go" ]
