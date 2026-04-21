# touch

opsxcli touch — 创建空文件或更新文件时间戳

## 用法

`opsxcli touch <file> [file...]`

## 说明

如果指定文件不存在，则创建空文件；如果文件已存在，则更新其访问和修改时间为当前时间。支持同时操作多个文件。

## 选项

无额外选项。

## 示例

创建单个空文件：

```bash
opsxcli touch newfile.txt
```

同时创建多个空文件：

```bash
opsxcli touch a.txt b.txt c.txt
```

更新已有文件的时间戳：

```bash
opsxcli touch existing.log
```
