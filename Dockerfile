# Multi-stage: static Go binary into a single distroless image.
# Root (not :nonroot) so the mounted docker.sock is readable — socket access
# is already host-root-equivalent per the project constitution.
FROM golang:1.24 AS build
WORKDIR /src
# Allow module toolchains to download if a dep requires a newer Go.
ENV GOTOOLCHAIN=auto
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /portside .

FROM gcr.io/distroless/static
COPY --from=build /portside /portside
EXPOSE 8888
ENTRYPOINT ["/portside"]
