FROM golang:1.19-alpine AS build
ARG version
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -mod=readonly -trimpath -ldflags "-s -w -X main.version=$version"

FROM alpine:3.17
COPY --from=build /src/bingo /usr/bin
CMD ["/usr/bin/bingo"]
