FROM golang:alpine AS dev-env

WORKDIR /usr/local/go/src/airman.com/airtask
COPY . /usr/local/go/src/airman.com/airtask

RUN apk update && apk upgrade && \
    apk add --no-cache bash git

RUN go get ./...

RUN go build -o dist/airtask &&\
    cp -f dist/airtask /usr/local/bin/ &&\
    cp -f dist/airtask.json /usr/local/etc/ &&\

RUN ls -l && ls -l dist

CMD ["/usr/local/bin/airtask", "-c", "/usr/local/etc/airtask.json" ]