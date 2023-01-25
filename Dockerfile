FROM golang:1.19 as  build
WORKDIR /root/
COPY ./go.mod .
COPY ./go.sum .
RUN go mod download
COPY ./main.go .
RUN CGO_ENABLED=0 go build ./

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=build /root/slack-anonymous-questions ./
CMD ["./slack-anonymous-questions"]
