#--------------
# Builder
#--------------
FROM golang:1.23-alpine as builder

RUN apk add build-base
COPY . /root/yrs
WORKDIR /root/yrs/web
RUN ["go", "build", "--tags", "fts5", "-o", "yrs-web", "."]

#--------------
# Runner
#--------------

FROM alpine:latest as runner

COPY --from=builder /root/yrs/web/ /opt/yrs

VOLUME [ "/data" ]

ENV GIN_MODE=release
ENV PORT=8080
ENV ROOT_URL=""
EXPOSE $PORT

WORKDIR /opt/yrs
ENTRYPOINT /opt/yrs/yrs-web \
    --config /opt/yrs/config/config.yml \
    --port $PORT \
    --address 0.0.0.0 \
    --root-url "$ROOT_URL"
