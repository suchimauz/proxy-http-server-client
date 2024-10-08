FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY .bin/app /root/app

EXPOSE 8080

CMD ["/root/app"]