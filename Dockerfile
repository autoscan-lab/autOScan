FROM golang:1.22
WORKDIR /app
COPY . .
RUN go build -o /autoscan-linux-amd64 ./cmd/autoscan

FROM scratch
COPY --from=0 /autoscan-linux-amd64 /
