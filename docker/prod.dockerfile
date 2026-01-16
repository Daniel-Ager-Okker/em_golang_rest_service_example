FROM golang:1.24.10-alpine3.21 AS build

RUN apk update && apk add make g++ sqlite

WORKDIR /rest_service_example

COPY . .

RUN make dependencies && make build

ENTRYPOINT [ "./dist/app" ]