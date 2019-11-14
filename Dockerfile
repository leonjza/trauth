FROM golang:alpine as build

LABEL maintainer="Leon Jacobs <leonja511@gmail.com>"

COPY . /src

WORKDIR /src
RUN go build -o trauth

# final image
FROM golang:alpine

COPY --from=build /src/trauth /

VOLUME ["/config"]

ENTRYPOINT ["/trauth"]
