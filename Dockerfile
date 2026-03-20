FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /icecube ./cmd/server

FROM alpine:3.23

RUN apk --no-cache add ca-certificates \
    jpegoptim \
    oxipng \
    pngquant \
    libwebp-tools \
    imagemagick

WORKDIR /app

COPY --from=builder /icecube .
COPY config/ config/

EXPOSE 3331

CMD ["./icecube"]
