FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG APP=./cmd/sentinel
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app ${APP}

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/app /app/sentinel
COPY config/config.yaml /etc/sentinel/config.yaml
ENV SENTINEL_CONFIG=/etc/sentinel/config.yaml
EXPOSE 8080
ENTRYPOINT ["/app/sentinel"]
