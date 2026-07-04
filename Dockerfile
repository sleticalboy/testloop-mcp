# -- build stage --
FROM golang:1.25-alpine AS builder

# tree-sitter 需要 CGO，安装 C 编译器
RUN apk add --no-cache gcc musl-dev

WORKDIR /build

# 先拷依赖文件，利用 Docker 层缓存
COPY go.mod go.sum ./
RUN go mod download

# 拷源码
COPY . .

# CGO 编译（tree-sitter 依赖 C 库）
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o testloop-mcp .

# -- runtime stage --
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /build/testloop-mcp /usr/local/bin/testloop-mcp

EXPOSE 8080

ENTRYPOINT ["testloop-mcp"]
CMD ["--transport", "http", "--addr", ":8080"]
