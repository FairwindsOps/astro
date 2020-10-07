FROM golang:1.14 AS build
LABEL maintainer="Micah Huber <micah@fairwinds.com>"
LABEL maintainer="Luke Reed <luke@fairwinds.com>"
LABEL maintainer="Bader Boland <bader@fairwinds.com>"
WORKDIR /go/src/github.com/fairwindsops/astro
ADD . /go/src/github.com/fairwindsops/astro

RUN GO111MODULE=on GOOS=linux GOARCH=amd64 go build -ldflags "-s -w"


FROM gcr.io/distroless/base
COPY --from=build /go/src/github.com/fairwindsops/astro/astro /astro
COPY --from=build /go/src/github.com/fairwindsops/astro/conf.yml /conf.yml
CMD ["./astro"]
