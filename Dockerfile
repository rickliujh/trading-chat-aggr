FROM golang:latest as build 

COPY . /goapp

WORKDIR /goapp

RUN make build

from scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /goapp/bin/server /

CMD ["/server"]
