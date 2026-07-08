FROM node:22-alpine AS frontend
WORKDIR /src/web-vue
COPY web-vue/package*.json ./
RUN npm ci
COPY web-vue/ ./
RUN npm run build

FROM golang:1.25-alpine AS backend
WORKDIR /src
RUN apk add --no-cache build-base
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/static/vue-app ./static/vue-app
RUN go build -o /out/scenic-guide .

FROM alpine:3.22
WORKDIR /app
RUN apk add --no-cache wget
RUN addgroup -S app && adduser -S app -G app
COPY --from=backend /out/scenic-guide ./scenic-guide
COPY configs/config.example.yaml ./configs/config.yaml
COPY knowledge ./knowledge
COPY --from=backend /src/static ./static
RUN mkdir -p /app/data && chown -R app:app /app
USER app
EXPOSE 8080
CMD ["./scenic-guide"]
