FROM node:22-alpine AS css
WORKDIR /app
COPY package.json .
RUN npm install
COPY web/ web/
RUN npx @tailwindcss/cli -i web/static/css/input.css -o web/static/css/output.css --minify

FROM golang:1.23-alpine AS build
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
COPY --from=css /app/web/static/css/output.css web/static/css/output.css
RUN go mod tidy && CGO_ENABLED=0 go build -ldflags="-s -w" -o /deployhub .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata docker-cli && \
    adduser -D -u 1001 deployhub && \
    mkdir -p /data && \
    chown deployhub:deployhub /data
COPY --from=build /deployhub /deployhub
USER deployhub
EXPOSE 8080
ENTRYPOINT ["/deployhub"]
