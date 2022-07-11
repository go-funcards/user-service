FROM golang:1.18.3-alpine as build

WORKDIR /app

COPY . .

RUN apk update \
	&& apk add --no-cache build-base ca-certificates \
	&& update-ca-certificates \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o release/ .

FROM scratch

LABEL org.opencontainers.image.authors="Igor Agapie <igoragapie@gmail.com>"
LABEL org.opencontainers.image.vendor="FunCards"

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /app/release /
COPY --from=build /app/config.yaml /config.yaml
COPY --from=build /app/proto /proto/

EXPOSE 80

CMD ["/user-service", "serve"]