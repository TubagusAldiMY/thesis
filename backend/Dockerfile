# Go API SPK Zakat: build statis, lalu satu image kecil berisi server + migrate + useradmin.
# Build dari folder App/backend:  docker build -t spkzakat-api backend
FROM golang:1.23-alpine AS build
# Toolchain dari image saja (jangan mengunduh versi Go lain diam-diam).
ENV GOTOOLCHAIN=local CGO_ENABLED=0
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -o /out/ ./cmd/server ./cmd/migrate ./cmd/useradmin

FROM alpine:3.20
# Akun non-root: proses API tidak butuh hak apa pun di container.
RUN adduser -D -u 10001 app
COPY --from=build /out/ /app/
USER app
EXPOSE 8090
# Skema database diterapkan dulu (idempoten), baru server; exec supaya SIGTERM
# sampai ke server dan graceful shutdown jalan.
CMD ["sh", "-c", "/app/migrate && exec /app/server"]
