FROM golang:1.24-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/yeomyeong ./cmd/server

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=build /out/yeomyeong /usr/local/bin/yeomyeong
COPY content ./content
COPY web ./web
COPY docs ./docs
EXPOSE 8080 4001
CMD ["yeomyeong"]
