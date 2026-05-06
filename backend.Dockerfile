FROM golang:1.22 AS build
WORKDIR /app
COPY . .
RUN go build -o server ./cmd/server

FROM gcr.io/distroless/base
WORKDIR /
COPY --from=build /app/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
