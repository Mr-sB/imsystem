# -I 自动引用关联文件
protoc -I ./pb --go_out=../ ./pb/*.proto