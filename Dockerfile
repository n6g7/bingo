FROM golang:1.19-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -mod=readonly -trimpath -ldflags "-s -w"

FROM alpine:3.17
COPY --from=build /src/bingo /usr/bin
CMD ["/usr/bin/bingo"]
