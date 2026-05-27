FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /builder .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /builder /builder
ENTRYPOINT ["/builder"]
