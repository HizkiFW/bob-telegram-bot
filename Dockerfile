# Image to build the go binary
FROM golang:alpine AS builder

# Install git and ca certificates
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

# Create user to run the service
ENV USER=appuser
ENV UID=10001
ENV APP_NAME=bob

# See https://stackoverflow.com/a/55757473/12429735
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

# Copy source code
RUN mkdir -p $GOPATH/src/$APP_NAME
WORKDIR $GOPATH/src/$APP_NAME
COPY . .

# Install dependencies
RUN go get -d -v

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/app

# Create the final docker image
FROM scratch

# Import the user and group files from the builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy the compiled executable
COPY --from=builder /go/bin/app /go/bin/app

# Set user and entry point
USER $USER:$USER
EXPOSE 9000
ENTRYPOINT ["/go/bin/app"]
