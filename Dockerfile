FROM golang:1.22-alpine AS build
WORKDIR /app
COPY . .
RUN go build -o /autoscan-linux-amd64 ./cmd/autoscan

FROM scratch
COPY --from=build /autoscan-linux-amd64 /
