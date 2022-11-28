#--------------
# Builder
#--------------
FROM golang:1.19-alpine as builder

RUN apk add git build-base
WORKDIR /root
RUN git clone https://github.com/miquelruiz/yrs.git
WORKDIR /root/yrs/web
RUN ["go", "build", "-o", "yrs-web", "."]

#--------------
# Runner
#--------------

FROM alpine:latest as runner

RUN ls -lah
COPY --from=builder /root/yrs/web/yrs-web /usr/bin
COPY --from=builder /root/yrs/web/config /etc/yrs

VOLUME [ "/data" ]

ENV PORT=8080
EXPOSE $PORT

ENTRYPOINT yrs-web --port $PORT --address 0.0.0.0
