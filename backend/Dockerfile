FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/deploypulse ./cmd/deploypulse

FROM alpine:3.21
RUN apk add --no-cache ca-certificates wget && adduser -D -H -u 10001 deploypulse
COPY --from=build /out/deploypulse /usr/local/bin/deploypulse
USER deploypulse
EXPOSE 8080
ENTRYPOINT ["deploypulse"]
