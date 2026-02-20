# ============================================================
# Stage 1: Build the xagent binary
# ============================================================
FROM golang:1.26.0-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN make build

# ============================================================
# Stage 2: Minimal runtime image
# ============================================================
FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata curl

# Copy binary
COPY --from=builder /src/build/xagent /usr/local/bin/xagent

# Create xagent home directory
RUN /usr/local/bin/xagent onboard

ENTRYPOINT ["xagent"]
CMD ["gateway"]
