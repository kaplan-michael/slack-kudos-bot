FROM docker.io/library/golang:1.24-alpine AS builder

#Copy the source code
WORKDIR /app
COPY . /app

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o slack-kudos .
RUN chmod +x /app/slack-kudos

# Build the final image with config files
FROM scratch
COPY --from=builder /app/slack-kudos /slack-kudos

EXPOSE 8080
CMD ["/slack-kudos"]
