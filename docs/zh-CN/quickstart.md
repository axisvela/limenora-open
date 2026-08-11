# 快速开始（中文）

## 安装

从 [发布说明](../dist/README.md) 下载最新二进制。

## 运行 Edge Gateway

```sh
./limenora-edge --port 10255 --data-dir ~/.limenora/edge
```

## 运行 Enterprise Gateway

```sh
./limenora-enterprise --port 10256 --data-dir ~/.limenora/enterprise
```

## 验证

```sh
curl http://127.0.0.1:10255/healthz
curl http://127.0.0.1:10255/v1/health
```

更多示例见 [示例索引](../examples.md)。
