FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

EXPOSE 8080

CMD ["./app"]