FROM alpine:3.23

RUN apk add --no-cache 'jpegoptim=~1.5' 'oxipng=~9.1' 'pngquant=~3.0' 'libwebp-tools=~1.6' 'imagemagick=~7.1'
