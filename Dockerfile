FROM golang:1.22-alpine AS backend-builder
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -o /open-panel ./cmd/server

FROM node:20-alpine AS frontend-builder
WORKDIR /src
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci
COPY frontend/ ./
RUN npx vite build --outDir dist

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /opt/open-panel
COPY --from=backend-builder /open-panel .
COPY --from=frontend-builder /src/dist ./web
ENV OPEN_PANEL_PORT=8888
ENV OPEN_PANEL_DATA=/opt/open-panel/data
ENV OPEN_PANEL_WEB=/opt/open-panel/web
EXPOSE 8888
VOLUME ["/opt/open-panel/data"]
ENTRYPOINT ["./open-panel"]
