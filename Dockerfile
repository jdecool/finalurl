FROM golang:alpine AS build-env
ARG src=/go/src/github.com/jdecool/finalurl
ADD . ${src}
RUN cd ${src}/cmd/finalurl \
    && go build

FROM alpine
WORKDIR /app
COPY --from=build-env /go/src/github.com/jdecool/finalurl/cmd/finalurl/finalurl /app
ENTRYPOINT [ "./finalurl" ]
