# Dd 工具

磁盘数据转换和复制工具。

## 使用

```bash
# 复制文件
opsxcli dd if=source.txt of=dest.txt

# 复制并转换
opsxcli dd if=/dev/zero of=test.img bs=1M count=100

# 创建镜像
opsxcli dd if=/dev/sda of=/path/to/image.img

# 烧录ISO到U盘
opsxcli dd if=ubuntu.iso of=/dev/sdb bs=4M
```

## 参数

- `if`: 输入文件
- `of`: 输出文件
- `bs`: 块大小
- `count`: 块数量
- `conv`: 转换选项

## 警告

此工具直接操作底层设备，请谨慎使用，避免数据丢失。
