FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server \
 && mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
COPY --from=build --chown=nonroot:nonroot /out/data /data
ENV HTTP_ADDR=:8080 \
    DATABASE_PATH=/data/app.db
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/server"]
