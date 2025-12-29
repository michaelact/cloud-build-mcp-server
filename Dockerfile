FROM golang AS build-env
COPY . /go-src/
WORKDIR /go-src/cmd/cloud-build-mcp-server
ENV CGO_ENABLED=0
RUN go build -o /go-app .

FROM gcr.io/distroless/base
COPY --from=build-env /go-app /
ENTRYPOINT ["/go-app", "--alsologtostderr", "--v=0"]
