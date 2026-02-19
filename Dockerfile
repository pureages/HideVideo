# 使用 Debian slim 基础镜像
FROM debian:stable-slim

# 设置工作目录
WORKDIR /app

# 安装 go-sqlite3 所需的 C 库
RUN apt-get update && \
    apt-get install -y libsqlite3-0 && \
    rm -rf /var/lib/apt/lists/*

# 拷贝可执行文件和前端、数据
COPY hidevideo .
COPY frontend ./frontend
COPY data ./data

# 确保可执行权限
RUN chmod +x hidevideo

# 暴露服务端口
EXPOSE 49377

# 启动命令
CMD ["./hidevideo"]
