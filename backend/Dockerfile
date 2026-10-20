FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api && CGO_ENABLED=0 go build -o /out/seed ./cmd/seed

FROM alpine:3.22
RUN adduser -D app
USER app
COPY --from=build /out/api /out/seed /usr/local/bin/
EXPOSE 8080
CMD ["api"]
