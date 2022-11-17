FROM golang:1.17.6-alpine AS builder
ARG SVC

WORKDIR /kv

COPY ./cmd/auth .

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && apk --no-cache add tzdata

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -ldflags "-s -w" -o build/kv-$SVC cmd/$SVC/main.go
RUN mv build/kv-$SVC /exe

FROM scratch

COPY --from=builder /exe /
COPY --from=builder /var/log /var/log

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=Asia/Shanghai

ENTRYPOINT ["/exe"]