FROM golang:1.20-alpine AS build
ARG version
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build \
  -mod=readonly \
  -trimpath \
  -ldflags "-s -w -X github.com/n6g7/bingo/cmd/bingo.version=$version" \
  ./cmd/bingo

FROM alpine:3.17
COPY --from=build /src/bingo /usr/bin
CMD ["/usr/bin/bingo"]
