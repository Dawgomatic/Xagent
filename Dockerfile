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
RUN go mod tidy
RUN make build

# ============================================================
# Stage 2: Minimal runtime image
# ============================================================
FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata curl

# Copy binary
COPY --from=builder /src/build/xagent /usr/local/bin/xagent

# SWE100821: --yes avoids interactive prompt in non-TTY Docker build context
RUN /usr/local/bin/xagent onboard --yes

ENTRYPOINT ["xagent"]
CMD ["gateway"]
