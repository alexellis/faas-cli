FROM golang:1.8.3 as builder

WORKDIR /go/src/github.com/alexellis/faas-cli
COPY . .

# Run a gofmt and exclude all vendored code.
RUN test -z "$(gofmt -l $(find . -type f -name '*.go' -not -path "./vendor/*"))"

RUN GIT_COMMIT=$(git rev-list -1 HEAD) \
 && CGO_ENABLED=0 GOOS=linux go build --ldflags "-s -w -X github.com/alexellis/faas-cli/commands.GitCommit=${GIT_COMMIT}" -a -installsuffix cgo -o faas-cli .
RUN go test ./...

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=0 /go/src/github.com/alexellis/faas-cli/faas-cli    .

CMD ["./faas-cli"]

