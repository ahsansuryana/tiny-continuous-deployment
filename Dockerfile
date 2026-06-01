FROM node:22-alpine AS css
WORKDIR /app
COPY package.json .
RUN npm install
COPY web/static/css/input.css web/static/css/input.css
RUN npx @tailwindcss/cli -i web/static/css/input.css -o web/static/css/output.css --minify

FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
COPY --from=css /app/web/static/css/output.css web/static/css/output.css
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /deployhub .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /deployhub /deployhub
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/deployhub"]
