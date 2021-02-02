FROM golang:alpine AS build

RUN apk add git
RUN mkdir -p /go/src/github.com/nibalizer/simpleFileSaver
WORKDIR /go/src/github.com/nibalizer/simpleFileSaver
COPY main.go go.* /go/src/github.com/nibalizer/simpleFileSaver/
RUN echo $GOPATH
RUN go get
RUN CGO_ENABLED=0 go build -o /bin/simpleFileSaver

FROM alpine
COPY --from=build /bin/simpleFileSaver /bin/simpleFileSaver
ENTRYPOINT ["/bin/simpleFileSaver"]
