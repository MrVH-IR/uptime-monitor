FROM node:26 AS frontend
WORKDIR /web
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /monitor ./cmd/monitor

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /monitor /monitor
WORKDIR /app
COPY --from=frontend /web/dist /app/frontend/dist
ENV HTTP_ADDR=0.0.0.0:8080
EXPOSE 8080
ENTRYPOINT ["/monitor"]
