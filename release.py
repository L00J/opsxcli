#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
上传 Release 到 Gitee（自动检测当前项目）
使用方法: 
  python3 release.py [版本号] [Gitee Token]
  
例如: 
  python3 release.py v1.0.1
  python3 release.py v1.0.1 your_gitee_token
  
Token 获取优先级:
  1. 命令行参数（第二个参数）
  2. 环境变量 GITEE_TOKEN
  3. ~/.netrc 文件中的 gitee.com 配置
  
或者配置 ~/.netrc:
  machine gitee.com
  login your_username
  password your_token
"""

import os
import sys
import json
import hashlib
import base64
import requests
import subprocess
import argparse
from pathlib import Path
from typing import Optional, Tuple, List, Dict

# 颜色输出
class Colors:
    RED = '\033[0;31m'
    GREEN = '\033[0;32m'
    YELLOW = '\033[1;33m'
    NC = '\033[0m'  # No Color

def info(msg: str):
    print(f"{Colors.GREEN}[INFO]{Colors.NC} {msg}")

def warn(msg: str):
    print(f"{Colors.YELLOW}[WARN]{Colors.NC} {msg}")

def error(msg: str):
    print(f"{Colors.RED}[ERROR]{Colors.NC} {msg}")

# Gitee API 基础 URL
API_BASE = "https://gitee.com/api/v5"

class GiteeReleaseUploader:
    def __init__(self, token: str):
        self.token = token
        self.headers = {
            "Authorization": f"token {token}",
            "User-Agent": "opsx-tools-release-uploader"
        }
    
    def get_release(self, repo: str, tag: str) -> Optional[dict]:
        """获取 Release 信息"""
        url = f"{API_BASE}/repos/{repo}/releases/tags/{tag}"
        try:
            response = requests.get(url, headers=self.headers, timeout=10)
            if response.status_code == 200:
                return response.json()
            elif response.status_code == 404:
                return None
            else:
                error(f"获取 Release 失败: {response.status_code} - {response.text}")
                return None
        except Exception as e:
            error(f"获取 Release 异常: {e}")
            return None
    
    def get_release_by_id(self, repo: str, release_id: int) -> Optional[dict]:
        """通过 Release ID 获取 Release 信息"""
        url = f"{API_BASE}/repos/{repo}/releases/{release_id}"
        try:
            response = requests.get(url, headers=self.headers, timeout=10)
            if response.status_code == 200:
                return response.json()
            else:
                warn(f"获取 Release 信息失败: {response.status_code}")
                return None
        except Exception as e:
            warn(f"获取 Release 信息异常: {e}")
            return None
    
    def get_release_attachments(self, repo: str, release_id: int) -> List[str]:
        """获取 Release 的附件列表（文件名）"""
        # Gitee API 获取 Release 详情时会包含附件信息
        # 我们需要通过 Release ID 获取完整的 Release 信息
        release_data = self.get_release_by_id(repo, release_id)
        if not release_data:
            return []
        
        # Gitee API 返回的附件信息在 assets 字段中
        attachments = []
        if "assets" in release_data:
            for asset in release_data["assets"]:
                if "name" in asset:
                    attachments.append(asset["name"])
        return attachments
    
    def generate_release_body(self, files: List[Path], tool_name: str, tag: str = None) -> str:
        """生成 Release 描述内容"""
        # 按平台分类文件
        platforms = {
            "Linux x86_64": [],
            "Linux ARM64": [],
            "macOS x86_64": [],
            "macOS ARM64": [],
            "Windows x86_64": [],
            "Windows x86": [],
        }
        
        for file_path in sorted(files):
            filename = file_path.name
            fl = filename.lower()
            if "linux" in fl and "x86_64" in fl:
                platforms["Linux x86_64"].append(filename)
            elif "linux" in fl and "aarch64" in fl:
                platforms["Linux ARM64"].append(filename)
            elif "darwin" in fl and "x86_64" in fl:
                platforms["macOS x86_64"].append(filename)
            elif "darwin" in fl and "arm64" in fl:
                platforms["macOS ARM64"].append(filename)
            elif "windows" in fl and "x86_64" in fl:
                platforms["Windows x86_64"].append(filename)
            elif "windows" in fl and ("i386" in fl or "386" in fl):
                platforms["Windows x86"].append(filename)
        
        # 生成描述
        lines = ["## 📦 跨平台发布\n"]
        
        # 快速安装（推荐）
        repo_url = f"https://gitee.com/opsx-tools/{tool_name}"
        releases_url = f"{repo_url}/releases"
        tag_str = tag if tag else "latest"
        
        lines.append("### 🚀 快速下载（推荐）\n")
        lines.append("自动检测平台并下载最新版本：\n")
        lines.append("```bash")
        tag_str = tag if tag else "latest"
        # 根据工具名确定文件名格式
        if tool_name == "opsxcli":
            # opsxcli 使用 opsxcli-{OS}-{ARCH} 格式
            lines.append(f"# 使用 wget")
            lines.append(f"wget \"https://gitee.com/opsx-tools/{tool_name}/releases/download/{tag_str}/{tool_name}-$(uname -s)-$(uname -m).tar.gz\"")
            lines.append("")
            lines.append(f"# 或使用 curl")
            lines.append(f"curl -L -o {tool_name}-$(uname -s)-$(uname -m).tar.gz \\")
            lines.append(f"  \"https://gitee.com/opsx-tools/{tool_name}/releases/download/{tag_str}/{tool_name}-$(uname -s)-$(uname -m).tar.gz\"")
        else:
            # 其他项目使用 opsx-{tool_name}-{OS}-{ARCH} 格式（首字母大写，与 uname 输出匹配）
            lines.append(f"# 使用 wget")
            lines.append(f"wget \"https://gitee.com/opsx-tools/{tool_name}/releases/download/{tag_str}/opsx-{tool_name}-$(uname -s)-$(uname -m).tar.gz\"")
            lines.append("")
            lines.append(f"# 或使用 curl")
            lines.append(f"curl -L -o opsx-{tool_name}-$(uname -s)-$(uname -m).tar.gz \\")
            lines.append(f"  \"https://gitee.com/opsx-tools/{tool_name}/releases/download/{tag_str}/opsx-{tool_name}-$(uname -s)-$(uname -m).tar.gz\"")
        lines.append("```\n")
        
        # 下载部分
        lines.append("### 📥 下载\n")
        for platform, filenames in platforms.items():
            if filenames:
                for filename in filenames:
                    if tag:
                        download_url = f"{repo_url}/releases/download/{tag}/{filename}"
                        lines.append(f"- **{platform}**: [{filename}]({download_url})")
                    else:
                        lines.append(f"- **{platform}**: `{filename}`")
        
        # 使用说明
        lines.append("\n### 📖 使用说明\n")
        lines.append("1. 根据您的操作系统和架构，下载对应的压缩包")
        lines.append("2. 解压后即可使用")
        lines.append("3. 详细文档请查看 [README.md](README.md)\n")
        
        # 校验文件说明
        lines.append("### 🔐 文件完整性验证\n")
        lines.append("下载 `checksums.txt` 文件后，可以使用以下命令验证：\n")
        lines.append("```bash")
        if tag:
            lines.append(f"# 下载校验文件")
            lines.append(f"curl -L -o checksums.txt \"{repo_url}/releases/download/{tag}/checksums.txt\"")
            lines.append("")
        lines.append("# Linux")
        lines.append("md5sum -c checksums.txt")
        lines.append("")
        lines.append("# macOS")
        lines.append("md5 -c checksums.txt")
        lines.append("```\n")
        
        # 手动安装示例
        lines.append("### 📦 手动安装\n")
        lines.append("```bash")
        lines.append("# Linux/macOS")
        if tool_name == "opsxcli":
            lines.append("tar -xzf opsxcli-*.tar.gz")
            lines.append("BINARY=$(tar tzf opsxcli-*.tar.gz | head -1)")
            lines.append("chmod +x \"$BINARY\"")
            lines.append("sudo mv \"$BINARY\" /usr/local/bin/opsxcli")
        else:
            lines.append("tar -xzf opsx-*-linux-amd64.tar.gz")
            lines.append("chmod +x opsx-*")
            lines.append("sudo mv opsx-* /usr/local/bin/")
        lines.append("")
        lines.append("# Windows")
        lines.append("# 解压 zip 文件后直接运行")
        lines.append("```")
        
        return "\n".join(lines)
    
    def create_release(self, repo: str, tag: str, files: List[Path] = None) -> Optional[dict]:
        """创建 Release"""
        url = f"{API_BASE}/repos/{repo}/releases"

        # 提取工具名称
        tool_name = repo.split("/")[-1]

        # 生成描述
        if files:
            body = self.generate_release_body(files, tool_name, tag)
        else:
            body = f"## 📦 Release {tag}\n\n详细内容请查看附件。"

        # 自动检测主分支（master 或 main）
        try:
            result = subprocess.run(
                ["git", "symbolic-ref", "refs/remotes/origin/HEAD"],
                capture_output=True,
                text=True,
                timeout=5
            )
            if result.returncode == 0:
                # 输出格式: refs/remotes/origin/master
                default_branch = result.stdout.strip().split("/")[-1]
            else:
                # 如果检测失败，尝试获取当前分支
                result = subprocess.run(
                    ["git", "branch", "--show-current"],
                    capture_output=True,
                    text=True,
                    timeout=5
                )
                if result.returncode == 0:
                    default_branch = result.stdout.strip()
                else:
                    default_branch = "master"  # 默认使用 master
        except:
            default_branch = "master"

        data = {
            "tag_name": tag,
            "name": f"Release {tag}",
            "body": body,
            "target_commitish": default_branch
        }

        try:
            response = requests.post(
                url,
                headers={**self.headers, "Content-Type": "application/json"},
                json=data,
                timeout=30
            )
            if response.status_code == 201:
                return response.json()
            else:
                error(f"创建 Release 失败: {response.status_code} - {response.text}")
                return None
        except Exception as e:
            error(f"创建 Release 异常: {e}")
            return None
    
    def update_release(self, repo: str, release_id: int, files: List[Path], tag: str = None) -> bool:
        """更新 Release 描述"""
        url = f"{API_BASE}/repos/{repo}/releases/{release_id}"
        
        # 提取工具名称
        tool_name = repo.split("/")[-1]
        
        # 如果没有提供 tag，从 Release 信息中获取
        if not tag:
            release = self.get_release_by_id(repo, release_id)
            if release:
                tag = release.get("tag_name", "")
        
        # 生成新的描述
        body = self.generate_release_body(files, tool_name, tag)
        
        # Gitee API 要求必须包含 tag_name 和 name
        data = {
            "tag_name": tag or "",
            "name": f"Release {tag}" if tag else "Release",
            "body": body
        }
        
        try:
            response = requests.patch(
                url,
                headers={**self.headers, "Content-Type": "application/json"},
                json=data,
                timeout=30
            )
            if response.status_code == 200:
                info("✓ Release 描述已更新")
                return True
            else:
                warn(f"更新 Release 描述失败: {response.status_code} - {response.text}")
                return False
        except Exception as e:
            warn(f"更新 Release 描述异常: {e}")
            return False
    
    def get_or_create_release(self, repo: str, tag: str, files: List[Path] = None) -> Tuple[Optional[int], bool]:
        """获取或创建 Release，返回 (Release ID, 是否为新创建)"""
        info(f"检查 Release: {repo} -> {tag}")
        
        # 先尝试获取
        release = self.get_release(repo, tag)
        if release and "id" in release:
            info(f"Release 已存在，ID: {release['id']}")
            return release["id"], False
        
        # 创建新的 Release
        info(f"创建 Release: {tag}")
        release = self.create_release(repo, tag, files)
        if release and "id" in release:
            info(f"Release 创建成功，ID: {release['id']}")
            return release["id"], True
        
        error("无法获取或创建 Release")
        return None, False
    
    def upload_attachment(self, repo: str, release_id: int, file_path: Path, existing_files: List[str] = None) -> bool:
        """上传附件到 Release"""
        filename = file_path.name
        
        # 检查文件是否已存在
        if existing_files and filename in existing_files:
            info(f"⏭  跳过（已存在）: {filename}")
            return True  # 返回 True 表示"已处理"，不需要重新上传
        
        info(f"上传附件: {filename}")
        
        # 检查文件是否存在
        if not file_path.exists():
            warn(f"文件不存在: {file_path}")
            return False
        
        # 检查文件大小（Gitee 限制 100M）
        file_size = file_path.stat().st_size
        max_size = 100 * 1024 * 1024  # 100MB
        
        if file_size > max_size:
            error(f"文件过大: {filename} ({file_size} bytes > 100MB)")
            return False
        
        # 上传文件
        url = f"{API_BASE}/repos/{repo}/releases/{release_id}/attach_files"
        
        try:
            with open(file_path, 'rb') as f:
                files = {'file': (filename, f, 'application/octet-stream')}
                response = requests.post(
                    url,
                    headers=self.headers,
                    files=files,
                    timeout=300  # 5分钟超时，大文件可能需要更长时间
                )
            
            if response.status_code == 201:
                info(f"✓ 上传成功: {filename}")
                return True
            elif response.status_code == 400:
                # 可能是文件已存在或其他错误
                response_text = response.text.lower()
                if "already" in response_text or "exist" in response_text or "重复" in response_text:
                    info(f"⏭  跳过（已存在）: {filename}")
                    return True
                else:
                    error(f"上传失败: {filename} - {response.status_code}")
                    warn(f"响应: {response.text}")
                    return False
            else:
                error(f"上传失败: {filename} - {response.status_code}")
                warn(f"响应: {response.text}")
                return False
        except Exception as e:
            error(f"上传异常: {filename} - {e}")
            return False
    
    def process_project(self, project_dir: str, repo: str, version: str) -> Tuple[int, int]:
        """处理单个项目"""
        # 检查项目目录（支持相对路径和绝对路径）
        project_path = Path(project_dir).resolve()
        if not project_path.exists():
            warn(f"项目目录不存在: {project_dir}，跳过")
            return 0, 0
        
        # 检查 dist 目录（在当前目录下搜索）
        dist_dir = project_path / "dist"
        if not dist_dir.exists():
            error(f"dist 目录不存在: {dist_dir}")
            error("请先运行构建脚本生成 releases 包")
            return 0, 0
        
        # 查找所有压缩包和校验文件
        files = list(dist_dir.glob("*.tar.gz")) + list(dist_dir.glob("*.zip"))
        checksums_file = dist_dir / "checksums.txt"
        
        if not files:
            warn(f"dist 目录中没有找到压缩包，跳过")
            return 0, 0
        
        # 如果存在 checksums.txt，也添加到上传列表
        if checksums_file.exists():
            files.append(checksums_file)
            info(f"找到 checksums.txt 文件")
        
        info(f"找到 {len(files)} 个文件")
        
        # 获取或创建 Release（如果是新创建的，已经包含描述）
        release_id, is_new = self.get_or_create_release(repo, version, files)
        if not release_id:
            error("无法获取 Release ID，跳过")
            return 0, 0
        
        # 获取已存在的附件列表（如果不是新创建的 Release）
        existing_files = []
        if not is_new:
            info("检查已存在的附件...")
            existing_files = self.get_release_attachments(repo, release_id)
            if existing_files:
                info(f"发现 {len(existing_files)} 个已存在的附件")
        
        # 上传所有文件
        success_count = 0
        fail_count = 0
        uploaded_files = []
        skipped_count = 0
        
        for file_path in files:
            result = self.upload_attachment(repo, release_id, file_path, existing_files)
            if result:
                success_count += 1
                # 如果文件已存在，不添加到 uploaded_files（不删除本地文件）
                if file_path.name not in existing_files:
                    uploaded_files.append(file_path)
                else:
                    skipped_count += 1
            else:
                fail_count += 1
        
        # 更新 Release 描述（使用所有文件列表，包括已存在的）
        # 无论是新创建还是已存在，都更新描述以确保包含最新的安装说明
        if files:
            info("更新 Release 描述...")
            self.update_release(repo, release_id, files, version)
        
        if skipped_count > 0:
            info(f"上传完成: 成功 {success_count} 个（跳过 {skipped_count} 个已存在的），失败 {fail_count} 个")
        else:
            info(f"上传完成: 成功 {success_count} 个，失败 {fail_count} 个")
        
        # 删除成功上传的文件（但保留 checksums.txt）
        if uploaded_files:
            info("清理已上传的文件...")
            deleted_count = 0
            for file_path in uploaded_files:
                # 保留 checksums.txt 文件，供用户下载验证
                if file_path.name == "checksums.txt":
                    info(f"  ⏭  保留: {file_path.name} (供用户下载验证)")
                    continue
                try:
                    file_path.unlink()
                    deleted_count += 1
                    info(f"  ✓ 已删除: {file_path.name}")
                except Exception as e:
                    warn(f"  删除文件失败: {file_path.name} - {e}")
            
            if deleted_count > 0:
                info(f"已删除 {deleted_count} 个已上传的文件")
            
            # 如果所有文件都上传成功，尝试删除 dist 目录（如果为空或只有 checksums.txt）
            if fail_count == 0:
                try:
                    # 检查 dist 目录是否为空或只有 checksums.txt
                    remaining_files = [f for f in dist_dir.glob("*") if f.name != "checksums.txt"]
                    if not remaining_files:
                        # 如果只有 checksums.txt，也删除它（因为已经上传到 Release）
                        checksums = dist_dir / "checksums.txt"
                        if checksums.exists():
                            checksums.unlink()
                        dist_dir.rmdir()
                        info(f"✓ 已清理空的 dist 目录")
                except Exception as e:
                    # 目录不为空或删除失败，忽略
                    pass
        
        if fail_count == 0:
            info(f"✓ 项目 {project_dir} 处理完成")
        else:
            warn(f"项目 {project_dir} 部分文件上传失败（失败的文件已保留）")
        
        return success_count, fail_count


# ═══════════════════════════════════════════════════════════
# GitHub Release 上传（只放预编译二进制，不放源码）
# ═══════════════════════════════════════════════════════════

GITHUB_API = "https://api.github.com"
GITHUB_REPO = "L00J/opsxcli"           # GitHub 发布仓库
HOMEBREW_TAP_REPO = "L00J/homebrew-tap" # Homebrew Tap 仓库
HOMEBREW_TAP_BRANCH = "main"


class GitHubReleaseUploader:
    """上传预编译二进制到 GitHub Release"""

    def __init__(self, token: str):
        self.token = token
        self.headers = {
            "Authorization": f"token {token}",
            "Accept": "application/vnd.github+json",
            "User-Agent": "opsx-tools-release-uploader",
        }

    # ── Release 管理 ─────────────────────────────────────

    def get_release(self, tag: str) -> Optional[dict]:
        url = f"{GITHUB_API}/repos/{GITHUB_REPO}/releases/tags/{tag}"
        try:
            r = requests.get(url, headers=self.headers, timeout=10)
            if r.status_code == 200:
                return r.json()
            return None
        except Exception as e:
            warn(f"GitHub: 获取 Release 异常: {e}")
            return None

    def create_release(self, tag: str, body: str = "") -> Optional[dict]:
        url = f"{GITHUB_API}/repos/{GITHUB_REPO}/releases"
        data = {
            "tag_name": tag,
            "name": f"Release {tag}",
            "body": body,
            "draft": False,
            "prerelease": False,
        }
        try:
            r = requests.post(url, headers={**self.headers, "Content-Type": "application/json"},
                              json=data, timeout=30)
            if r.status_code == 201:
                info(f"GitHub: Release {tag} 创建成功")
                return r.json()
            error(f"GitHub: 创建 Release 失败: {r.status_code} - {r.text[:200]}")
            return None
        except Exception as e:
            error(f"GitHub: 创建 Release 异常: {e}")
            return None

    def get_or_create_release(self, tag: str) -> Optional[dict]:
        release = self.get_release(tag)
        if release:
            info(f"GitHub: Release 已存在 (ID: {release['id']})")
            return release
        return self.create_release(tag, self._generate_body(tag))

    # ── 附件上传 ─────────────────────────────────────────

    def upload_asset(self, release: dict, file_path: Path) -> bool:
        upload_url = release["upload_url"].split("{")[0]  # 去掉 {?name,label}
        filename = file_path.name

        # 检查是否已存在
        existing = [a["name"] for a in release.get("assets", [])]
        if filename in existing:
            info(f"GitHub: ⏭ 跳过（已存在）: {filename}")
            return True

        info(f"GitHub: 上传 {filename} ...")
        try:
            with open(file_path, "rb") as f:
                r = requests.post(
                    upload_url,
                    headers={**self.headers, "Content-Type": "application/octet-stream"},
                    params={"name": filename},
                    data=f,
                    timeout=300,
                )
            if r.status_code == 201:
                info(f"GitHub: ✓ 上传成功: {filename}")
                return True
            error(f"GitHub: 上传失败 {filename}: {r.status_code}")
            return False
        except Exception as e:
            error(f"GitHub: 上传异常 {filename}: {e}")
            return False

    # ── Release Body 生成 ────────────────────────────────

    def _generate_body(self, tag: str) -> str:
        lines = [
            f"## OpsXCLI {tag}",
            "",
            "### 🚀 快速安装",
            "",
            "**Homebrew (macOS/Linux)**:",
            "```bash",
            "brew tap L00J/tap",
            "brew install opsxcli",
            "```",
            "",
            "**一键脚本 (Linux/macOS)**:",
            "```bash",
            f"curl -fsSL https://github.com/{GITHUB_REPO}/releases/latest/download/install.sh | bash",
            "```",
            "",
            "### 📥 下载",
            "",
            "| 平台 | 架构 | 文件 |",
            "|------|------|------|",
            "| Linux | x86_64 | opsxcli-Linux-x86_64.tar.gz |",
            "| Linux | arm64 | opsxcli-Linux-aarch64.tar.gz |",
            "| macOS | x86_64 | opsxcli-Darwin-x86_64.tar.gz |",
            "| macOS | arm64 | opsxcli-Darwin-arm64.tar.gz |",
            "| Windows | x86_64 | opsxcli-Windows-x86_64.zip |",
            "",
            "### 📦 手动安装",
            "```bash",
            "tar -xzf opsxcli-*.tar.gz",
            "BINARY=$(tar tzf opsxcli-*.tar.gz | head -1)",
            "chmod +x \"$BINARY\"",
            "sudo mv \"$BINARY\" /usr/local/bin/opsxcli",
            "```",
            "",
            "---",
            f"**完整更新日志**: https://gitee.com/opsx-tools/opsxcli/blob/master/CHANGELOG.md",
        ]
        return "\n".join(lines)

    # ── 完整发布流程 ─────────────────────────────────────

    def publish(self, version: str, dist_dir: Path) -> Tuple[int, int]:
        """发布到 GitHub Release，返回 (成功数, 失败数)"""
        # 收集文件
        files = sorted(dist_dir.glob("*.tar.gz")) + sorted(dist_dir.glob("*.zip"))
        checksums = dist_dir / "checksums.txt"
        if checksums.exists():
            files.append(checksums)
        if not files:
            warn("GitHub: dist/ 中没有找到构建产物")
            return 0, 0

        info(f"GitHub: 找到 {len(files)} 个文件")

        # 确保 tag 存在
        self._ensure_tag(version)

        # 创建/获取 Release
        release = self.get_or_create_release(version)
        if not release:
            return 0, len(files)

        # 上传
        ok, fail = 0, 0
        for fp in files:
            if self.upload_asset(release, fp):
                ok += 1
            else:
                fail += 1

        info(f"GitHub: 上传完成 — 成功 {ok}, 失败 {fail}")
        return ok, fail

    def _ensure_tag(self, tag: str):
        """确保 GitHub 上存在该 tag（通过 push github）"""
        try:
            result = subprocess.run(
                ["git", "ls-remote", "--tags", f"https://github.com/{GITHUB_REPO}.git", tag],
                capture_output=True, text=True, timeout=10,
            )
            if tag in result.stdout:
                info(f"GitHub: tag {tag} 已存在")
                return
            # 推送 tag
            subprocess.run(["git", "tag", tag], capture_output=True, timeout=5)
            subprocess.run(["git", "push", "github", tag], capture_output=True, text=True, timeout=30)
            info(f"GitHub: 已推送 tag {tag}")
        except Exception as e:
            warn(f"GitHub: tag 操作失败: {e}")


# ═══════════════════════════════════════════════════════════
# Homebrew Formula 生成 + 推送
# ═══════════════════════════════════════════════════════════

class HomebrewFormulaPusher:
    """生成只含预编译二进制的 Homebrew Formula，推送到 L00J/homebrew-tap"""

    def __init__(self, token: str):
        self.token = token
        self.headers = {
            "Authorization": f"token {token}",
            "Accept": "application/vnd.github+json",
            "User-Agent": "opsx-tools-release-uploader",
        }

    def _sha256(self, file_path: Path) -> str:
        """计算文件 SHA256"""
        import hashlib
        h = hashlib.sha256()
        with open(file_path, "rb") as f:
            for chunk in iter(lambda: f.read(8192), b""):
                h.update(chunk)
        return h.hexdigest()

    def generate_formula(self, version: str, dist_dir: Path) -> str:
        """生成 Homebrew Formula（只含预编译二进制下载，无源码）"""
        # 找到 darwin-arm64 包作为 macOS 默认（Homebrew 最常用）
        # 需要为每个平台生成 resource
        tag = version
        ver = version.lstrip("v")

        # 收集所有平台的包信息
        resources = []
        for f in sorted(dist_dir.glob("*.tar.gz")):
            name = f.name
            sha = self._sha256(f)
            url = f"https://github.com/{GITHUB_REPO}/releases/download/{tag}/{name}"

            # 解析平台
            fl = name.lower()
            if "darwin" in fl and "arm64" in fl:
                plat, arch = "darwin", "arm64"
            elif "darwin" in fl and "x86_64" in fl:
                plat, arch = "darwin", "x86_64"
            elif "linux" in fl and "x86_64" in fl:
                plat, arch = "linux", "x86_64"
            elif "linux" in fl and "aarch64" in fl:
                plat, arch = "linux", "aarch64"
            else:
                continue

            resources.append({
                "platform": plat,
                "arch": arch,
                "url": url,
                "sha256": sha,
                "filename": name,
            })

        if not resources:
            error("Homebrew: 未找到任何构建产物")
            return ""

        # 使用 darwin-arm64 作为主 URL
        main = None
        for r in resources:
            if r["platform"] == "darwin" and r["arch"] == "arm64":
                main = r
                break
        if not main:
            main = resources[0]

        # 生成 Formula
        lines = [
            "# 自动生成 — 请勿手动编辑",
            f"# 由 release.py 从预编译二进制生成（不含源码）",
            f"# 版本: {version}",
            "",
            'class Opsxcli < Formula',
            f'  desc "OpsXCLI - DevOps CLI toolkit with AI agent, TUI dashboard"',
            f'  homepage "https://gitee.com/opsx-tools/opsxcli"',
            f'  url "{main["url"]}"',
            f'  sha256 "{main["sha256"]}"',
            f'  version "{ver}"',
            "",
            '  def install',
            '    bin.install "opsxcli"',
            '  end',
            "",
            '  test do',
            '    system "#{bin}/opsxcli", "--help"',
            '  end',
            "end",
            "",
        ]
        return "\n".join(lines)

    def push_formula(self, version: str, dist_dir: Path) -> bool:
        """推送 Formula 到 L00J/homebrew-tap 仓库"""
        formula_content = self.generate_formula(version, dist_dir)
        if not formula_content:
            return False

        formula_path = "Formula/opsxcli.rb"
        api_url = f"{GITHUB_API}/repos/{HOMEBREW_TAP_REPO}/contents/{formula_path}"

        # 获取当前文件 SHA（用于更新）
        existing_sha = None
        try:
            r = requests.get(api_url, headers=self.headers, params={"ref": HOMEBREW_TAP_BRANCH}, timeout=10)
            if r.status_code == 200:
                existing_sha = r.json().get("sha")
                info(f"Homebrew: Formula 已存在，将更新 (sha: {existing_sha[:8]}...)")
        except Exception:
            pass

        # 上传/更新文件
        import base64
        encoded = base64.b64encode(formula_content.encode()).decode()

        data = {
            "message": f"opsxcli {version} — auto-update formula",
            "content": encoded,
            "branch": HOMEBREW_TAP_BRANCH,
        }
        if existing_sha:
            data["sha"] = existing_sha

        try:
            r = requests.put(api_url, headers=self.headers, json=data, timeout=30)
            if r.status_code in (200, 201):
                info(f"Homebrew: ✓ Formula 推送成功 — {formula_path}")
                info(f"  安装方式: brew tap L00J/tap && brew install opsxcli")
                return True
            else:
                error(f"Homebrew: Formula 推送失败: {r.status_code} - {r.text[:200]}")
                return False
        except Exception as e:
            error(f"Homebrew: Formula 推送异常: {e}")
            return False


# ─────────────────────────────────────────────────────
# GitHub Release Uploader
# ─────────────────────────────────────────────────────

GITHUB_API = "https://api.github.com"
GITHUB_UPLOAD = "https://uploads.github.com"

class GitHubReleaseUploader:
    """上传预编译二进制到 GitHub Release（不放源码）"""

    def __init__(self, token: str):
        self.token = token
        self.headers = {
            "Authorization": f"token {token}",
            "Accept": "application/vnd.github+json",
            "User-Agent": "opsx-tools-release-uploader",
            "X-GitHub-Api-Version": "2022-11-28",
        }

    def get_release(self, repo: str, tag: str) -> Optional[dict]:
        """获取 Release 信息"""
        url = f"{GITHUB_API}/repos/{repo}/releases/tags/{tag}"
        try:
            resp = requests.get(url, headers=self.headers, timeout=10)
            if resp.status_code == 200:
                return resp.json()
            elif resp.status_code == 404:
                return None
            else:
                error(f"GitHub 获取 Release 失败: {resp.status_code} - {resp.text[:200]}")
                return None
        except Exception as e:
            error(f"GitHub 获取 Release 异常: {e}")
            return None

    def create_release(self, repo: str, tag: str, body: str = "") -> Optional[dict]:
        """创建 GitHub Release"""
        url = f"{GITHUB_API}/repos/{repo}/releases"
        data = {
            "tag_name": tag,
            "name": f"Release {tag}",
            "body": body,
            "draft": False,
            "prerelease": False,
        }
        try:
            resp = requests.post(
                url,
                headers={**self.headers, "Content-Type": "application/json"},
                json=data,
                timeout=30,
            )
            if resp.status_code == 201:
                info(f"✓ GitHub Release 创建成功: {tag}")
                return resp.json()
            else:
                error(f"GitHub 创建 Release 失败: {resp.status_code} - {resp.text[:200]}")
                return None
        except Exception as e:
            error(f"GitHub 创建 Release 异常: {e}")
            return None

    def get_or_create_release(self, repo: str, tag: str, body: str = "") -> Tuple[Optional[int], bool]:
        """获取或创建 Release，返回 (Release ID, 是否新创建)"""
        info(f"[GitHub] 检查 Release: {repo} -> {tag}")
        release = self.get_release(repo, tag)
        if release and "id" in release:
            info(f"[GitHub] Release 已存在，ID: {release['id']}")
            return release["id"], False

        info(f"[GitHub] 创建 Release: {tag}")
        release = self.create_release(repo, tag, body)
        if release and "id" in release:
            return release["id"], True

        error("[GitHub] 无法获取或创建 Release")
        return None, False

    def list_assets(self, repo: str, release_id: int) -> List[str]:
        """获取 Release 已有 asset 文件名列表"""
        url = f"{GITHUB_API}/repos/{repo}/releases/{release_id}/assets"
        try:
            resp = requests.get(url, headers=self.headers, timeout=10)
            if resp.status_code == 200:
                return [a["name"] for a in resp.json()]
            return []
        except Exception:
            return []

    def upload_asset(self, repo: str, release_id: int, file_path: Path,
                     existing: List[str] = None) -> bool:
        """上传单个 asset 到 GitHub Release"""
        filename = file_path.name

        if existing and filename in existing:
            info(f"  [GitHub] ⏭  跳过（已存在）: {filename}")
            return True

        if not file_path.exists():
            warn(f"[GitHub] 文件不存在: {file_path}")
            return False

        info(f"  [GitHub] 上传: {filename} ({file_path.stat().st_size / 1024 / 1024:.1f} MB)")

        url = (
            f"{GITHUB_UPLOAD}/repos/{repo}/releases/{release_id}/assets"
            f"?name={filename}"
        )
        try:
            with open(file_path, "rb") as f:
                resp = requests.post(
                    url,
                    headers={
                        **self.headers,
                        "Content-Type": "application/octet-stream",
                    },
                    data=f,
                    timeout=600,
                )
            if resp.status_code in (201, 200):
                info(f"  [GitHub] ✓ 上传成功: {filename}")
                return True
            else:
                error(f"  [GitHub] 上传失败: {filename} - {resp.status_code}")
                warn(f"  响应: {resp.text[:200]}")
                return False
        except Exception as e:
            error(f"  [GitHub] 上传异常: {filename} - {e}")
            return False

    def generate_github_body(self, files: List[Path], tool_name: str, tag: str) -> str:
        """生成 GitHub Release 描述（含 Homebrew 安装说明）"""
        lines = [f"## 📦 {tool_name} {tag}\n"]
        lines.append("### 🚀 快速安装\n")
        lines.append("```bash")
        lines.append("# Homebrew (macOS / Linux)")
        lines.append("brew tap L00J/tap")
        lines.append(f"brew install {tool_name}")
        lines.append("")
        lines.append("# 或一键安装脚本")
        lines.append(f"curl -fsSL https://raw.githubusercontent.com/L00J/{tool_name}/master/install.sh | bash")
        lines.append("```\n")

        lines.append("### 📥 下载\n")
        for f in sorted(files):
            if f.name == "checksums.txt":
                continue
            fl = f.name.lower()
            if "linux" in fl and "x86_64" in fl:
                plat = "Linux x86_64"
            elif "linux" in fl and ("aarch64" in fl or "arm64" in fl):
                plat = "Linux ARM64"
            elif "darwin" in fl and "x86_64" in fl:
                plat = "macOS x86_64"
            elif "darwin" in fl and "arm64" in fl:
                plat = "macOS ARM64"
            elif "windows" in fl:
                plat = "Windows"
            else:
                plat = ""
            if plat:
                url = f"https://github.com/L00J/{tool_name}/releases/download/{tag}/{f.name}"
                lines.append(f"- **{plat}**: [{f.name}]({url})")

        lines.append("\n### 📖 使用说明\n")
        lines.append("```bash")
        lines.append("tar -xzf opsxcli-*.tar.gz")
        lines.append("BINARY=$(tar tzf opsxcli-*.tar.gz | head -1)")
        lines.append("chmod +x \"$BINARY\"")
        lines.append("sudo mv \"$BINARY\" /usr/local/bin/opsxcli")
        lines.append("```")

        return "\n".join(lines)

    def process_project(self, repo: str, version: str, dist_dir: Path,
                        gitee_uploader: 'GiteeReleaseUploader' = None) -> Tuple[int, int]:
        """处理 GitHub Release 上传"""
        files = list(dist_dir.glob("*.tar.gz")) + list(dist_dir.glob("*.zip"))
        checksums = dist_dir / "checksums.txt"
        if checksums.exists():
            files.append(checksums)

        if not files:
            warn("[GitHub] dist 目录中没有文件，跳过")
            return 0, 0

        tool_name = repo.split("/")[-1]
        body = self.generate_github_body(files, tool_name, version)

        release_id, is_new = self.get_or_create_release(repo, version, body)
        if not release_id:
            return 0, len(files)

        existing = [] if is_new else self.list_assets(repo, release_id)

        success, fail = 0, 0
        for fp in files:
            if self.upload_asset(repo, release_id, fp, existing):
                success += 1
            else:
                fail += 1

        # 更新描述（确保包含最新信息）
        if not is_new and files:
            url = f"{GITHUB_API}/repos/{repo}/releases/{release_id}"
            try:
                requests.patch(
                    url,
                    headers={**self.headers, "Content-Type": "application/json"},
                    json={"body": body},
                    timeout=15,
                )
            except Exception:
                pass

        info(f"[GitHub] 上传完成: 成功 {success}，失败 {fail}")
        return success, fail


# ─────────────────────────────────────────────────────
# Homebrew Formula Generator
# ─────────────────────────────────────────────────────

HOMEBREW_TAP_REPO = "L00J/homebrew-tap"
HOMEBREW_TOOL_REPO = "L00J/opsxcli"  # GitHub repo for download URLs

# 平台映射: (文件名关键词, Homebrew 条件)
PLATFORM_MAP = [
    ("darwin-arm64",  "macos", "arm64",  "Hardware::CPU.arm?"),
    ("darwin-x86_64", "macos", "x86_64", "Hardware::CPU.intel?"),
    ("linux-aarch64", "linux", "arm64",  "Hardware::CPU.arm?"),
    ("linux-arm64",   "linux", "arm64",  "Hardware::CPU.arm?"),
    ("linux-x86_64",  "linux", "x86_64", "Hardware::CPU.intel?"),
]


class HomebrewFormulaGenerator:
    """生成 Homebrew Formula 并推送到 homebrew-tap 仓库"""

    def __init__(self, token: str, tap_repo: str = HOMEBREW_TAP_REPO,
                 tool_repo: str = HOMEBREW_TOOL_REPO):
        self.token = token
        self.tap_repo = tap_repo
        self.tool_repo = tool_repo
        self.headers = {
            "Authorization": f"token {token}",
            "Accept": "application/vnd.github+json",
            "User-Agent": "opsx-tools-release-uploader",
            "X-GitHub-Api-Version": "2022-11-28",
        }

    def _sha256(self, file_path: Path) -> str:
        """计算文件 SHA256"""
        h = hashlib.sha256()
        with open(file_path, "rb") as f:
            for chunk in iter(lambda: f.read(8192), b""):
                h.update(chunk)
        return h.hexdigest()

    def _find_files(self, dist_dir: Path) -> Dict[str, Path]:
        """从 dist/ 中找到各平台的 tar.gz 文件"""
        found = {}
        for fp in dist_dir.glob("*.tar.gz"):
            fl = fp.name.lower()
            for keyword, os_name, arch, condition in PLATFORM_MAP:
                if keyword in fl:
                    found[keyword] = fp
                    break
        return found

    def generate_formula(self, version: str, dist_dir: Path, tool_name: str = "opsxcli") -> str:
        """生成 Homebrew Formula 内容（多平台二进制）"""
        files = self._find_files(dist_dir)
        if not files:
            warn("[Homebrew] 未找到任何平台二进制文件")
            return ""

        version_clean = version.lstrip("v")
        base_url = f"https://github.com/{self.tool_repo}/releases/download/{version}"

        # 计算 SHA256
        sha_map = {}
        for keyword, fp in files.items():
            sha_map[keyword] = self._sha256(fp)
            info(f"  [Homebrew] {fp.name}: sha256={sha_map[keyword][:16]}...")

        # 生成 Formula
        lines = [
            f'# Auto-generated by release.py — {tool_name} {version}',
            f'class {tool_name.capitalize()} < Formula',
            f'  desc "OpsXCLI — 运维超级工具集 (DevOps CLI)"',
            f'  homepage "https://github.com/{self.tool_repo}"',
            f'  url "{base_url}/opsxcli-Darwin-arm64.tar.gz"',
            f'  sha256 "{sha_map.get("darwin-arm64", "PLACEHOLDER")}"',
            f'  version "{version_clean}"',
            f'',
            f'  def install',
            f'    bin.install "{tool_name}"',
            f'  end',
            f'',
            f'  test do',
            f'    assert_match "{version_clean}", shell_output("#{bin}/{tool_name} --version 2>&1 || true")',
            f'  end',
            f'end',
        ]

        # 如果有多平台，使用 on_macos/on_linux 条件块
        has_macos = any(k.startswith("darwin") for k in files)
        has_linux = any(k.startswith("linux") for k in files)

        if has_macos and has_linux:
            lines = [
                f'# Auto-generated by release.py — {tool_name} {version}',
                f'class {tool_name.capitalize()} < Formula',
                f'  desc "OpsXCLI — 运维超级工具集 (DevOps CLI)"',
                f'  homepage "https://github.com/{self.tool_repo}"',
                f'  version "{version_clean}"',
                f'',
            ]
            # macOS
            if "darwin-arm64" in sha_map or "darwin-x86_64" in sha_map:
                lines.append(f'  on_macos do')
                if "darwin-arm64" in sha_map:
                    lines.append(f'    if Hardware::CPU.arm?')
                    lines.append(f'      url "{base_url}/opsxcli-Darwin-arm64.tar.gz"')
                    lines.append(f'      sha256 "{sha_map["darwin-arm64"]}"')
                    lines.append(f'    end')
                if "darwin-x86_64" in sha_map:
                    lines.append(f'    if Hardware::CPU.intel?')
                    lines.append(f'      url "{base_url}/opsxcli-Darwin-x86_64.tar.gz"')
                    lines.append(f'      sha256 "{sha_map["darwin-x86_64"]}"')
                    lines.append(f'    end')
                lines.append(f'  end')
                lines.append(f'')

            # Linux
            if any(k.startswith("linux") for k in sha_map):
                lines.append(f'  on_linux do')
                if "linux-aarch64" in sha_map:
                    lines.append(f'    if Hardware::CPU.arm?')
                    lines.append(f'      url "{base_url}/opsxcli-Linux-aarch64.tar.gz"')
                    lines.append(f'      sha256 "{sha_map["linux-aarch64"]}"')
                    lines.append(f'    end')
                if "linux-arm64" in sha_map and "linux-aarch64" not in sha_map:
                    lines.append(f'    if Hardware::CPU.arm?')
                    lines.append(f'      url "{base_url}/opsxcli-Linux-arm64.tar.gz"')
                    lines.append(f'      sha256 "{sha_map["linux-arm64"]}"')
                    lines.append(f'    end')
                if "linux-x86_64" in sha_map:
                    lines.append(f'    if Hardware::CPU.intel?')
                    lines.append(f'      url "{base_url}/opsxcli-Linux-x86_64.tar.gz"')
                    lines.append(f'      sha256 "{sha_map["linux-x86_64"]}"')
                    lines.append(f'    end')
                lines.append(f'  end')
                lines.append(f'')

            lines.extend([
                f'  def install',
                f'    bin.install "{tool_name}"',
                f'  end',
                f'',
                f'  test do',
                f'    assert_match "{version_clean}", shell_output("#{bin}/{tool_name} --version 2>&1 || true")',
                f'  end',
                f'end',
            ])

        return "\n".join(lines)

    def push_formula(self, version: str, dist_dir: Path,
                     tool_name: str = "opsxcli") -> bool:
        """生成 Formula 并推送到 homebrew-tap 仓库"""
        info(f"[Homebrew] 生成 Formula: {tool_name} {version}")
        content = self.generate_formula(version, dist_dir, tool_name)
        if not content:
            warn("[Homebrew] Formula 生成失败，跳过推送")
            return False

        formula_path = f"Formula/{tool_name}.rb"

        # 检查是否已存在
        url = f"{GITHUB_API}/repos/{self.tap_repo}/contents/{formula_path}"
        existing_sha = None
        try:
            resp = requests.get(url, headers=self.headers, timeout=10)
            if resp.status_code == 200:
                existing_sha = resp.json().get("sha")
                info(f"[Homebrew] Formula 已存在，将更新 (sha={existing_sha[:8]}...)")
        except Exception:
            pass

        # 推送
        encoded = base64.b64encode(content.encode()).decode()
        data = {
            "message": f"auto: update {tool_name} to {version}",
            "content": encoded,
            "branch": "main",
        }
        if existing_sha:
            data["sha"] = existing_sha

        try:
            resp = requests.put(
                url,
                headers={**self.headers, "Content-Type": "application/json"},
                json=data,
                timeout=30,
            )
            if resp.status_code in (200, 201):
                info(f"[Homebrew] ✓ Formula 推送成功: {formula_path}")
                info(f"[Homebrew] 查看: https://github.com/{self.tap_repo}/blob/main/{formula_path}")
                return True
            else:
                error(f"[Homebrew] 推送失败: {resp.status_code}")
                warn(f"  响应: {resp.text[:300]}")
                return False
        except Exception as e:
            error(f"[Homebrew] 推送异常: {e}")
            return False


def get_version() -> str:
    """获取版本号（纯 tag，不带 commit hash）"""
    if len(sys.argv) > 1:
        version = sys.argv[1]
        # 如果版本号不是以 v 开头，自动添加
        if not version.startswith("v"):
            version = f"v{version}"
        return version
    
    # 尝试从 git 获取最新的 tag（只获取 tag，不带 commit hash）
    try:
        import subprocess
        result = subprocess.run(
            ["git", "describe", "--tags", "--abbrev=0"],
            capture_output=True,
            text=True,
            timeout=5
        )
        if result.returncode == 0:
            return result.stdout.strip()
    except:
        pass
    
    return "dev"


def get_gitee_token() -> Optional[str]:
    """获取 Gitee Token（优先级：命令行参数 > 环境变量 > ~/.netrc）"""
    # 1. 命令行参数（第二个参数）
    if len(sys.argv) > 2:
        token = sys.argv[2]
        if token:
            return token
    
    # 2. 环境变量
    token = os.environ.get("GITEE_TOKEN")
    if token:
        return token
    
    # 3. 从 ~/.netrc 读取（手动解析，避免格式问题）
    netrc_path = Path.home() / ".netrc"
    if netrc_path.exists():
        try:
            # 先尝试使用 netrc 库
            import netrc
            nrc = netrc.netrc(str(netrc_path))
            auth = nrc.authenticators("gitee.com")
            if auth:
                token = auth[2]
                if token:
                    info("从 ~/.netrc 读取 Gitee Token")
                    return token
        except (ImportError, OSError, netrc.NetrcParseError):
            # netrc 库解析失败，尝试手动解析
            pass
        except Exception:
            pass
        
        # 手动解析（支持单行格式：machine gitee.com login user password token）
        try:
            with open(netrc_path, 'r') as f:
                for line in f:
                    line = line.strip()
                    if not line or line.startswith('#'):
                        continue
                    # 查找包含 gitee.com 的行
                    if 'gitee.com' in line.lower():
                        parts = line.split()
                        # 解析格式: machine gitee.com login attacker password token
                        for i, part in enumerate(parts):
                            if part.lower() == 'password' and i + 1 < len(parts):
                                token = parts[i + 1]
                                if token:
                                    info("从 ~/.netrc 读取 Gitee Token（手动解析）")
                                    return token
        except Exception:
            pass
    
    return None


def get_github_token() -> Optional[str]:
    """获取 GitHub Token（优先级：环境变量 > 命令行第三个参数）"""
    token = os.environ.get("GITHUB_TOKEN") or os.environ.get("HOMEBREW_TAP_TOKEN")
    if token:
        return token
    # 复用第三个命令行参数
    if len(sys.argv) > 3:
        return sys.argv[3]
    return None


def detect_current_project() -> Tuple[Optional[str], Optional[str]]:
    """自动检测当前项目目录和仓库名
    返回: (项目目录名, 仓库路径，如 opsx-tools/opsxcli)
    """
    # 获取当前工作目录
    current_dir = Path.cwd()
    project_name = current_dir.name
    
    # 尝试从 git remote 获取仓库信息
    repo_path = None
    try:
        result = subprocess.run(
            ["git", "remote", "get-url", "origin"],
            capture_output=True,
            text=True,
            timeout=5,
            cwd=current_dir
        )
        if result.returncode == 0:
            remote_url = result.stdout.strip()
            # 解析 git URL: https://gitee.com/opsx-tools/opsxcli.git
            # 或 git@gitee.com:opsx-tools/opsxcli.git
            if "gitee.com" in remote_url:
                if remote_url.startswith("http"):
                    # https://gitee.com/opsx-tools/opsxcli.git
                    parts = remote_url.replace(".git", "").split("/")
                    if len(parts) >= 2:
                        repo_path = f"{parts[-2]}/{parts[-1]}"
                elif remote_url.startswith("git@"):
                    # git@gitee.com:opsx-tools/opsxcli.git
                    parts = remote_url.replace(".git", "").split(":")
                    if len(parts) >= 2:
                        repo_path = parts[-1]
    except Exception:
        pass
    
    # 如果无法从 git 获取，使用默认格式
    if not repo_path:
        repo_path = f"opsx-tools/{project_name}"
    
    return project_name, repo_path


def main():
    """主流程：Gitee Release → GitHub Release → Homebrew Formula"""
    version = get_version()
    gitee_token = get_gitee_token()
    github_token = get_github_token()

    if not gitee_token:
        error("请提供 Gitee Token")
        print("\n使用方法:")
        print(f"  {sys.argv[0]} [版本号] [Gitee Token] [GitHub Token]")
        print("\n示例:")
        print(f"  {sys.argv[0]} v1.0.1")
        print(f"  {sys.argv[0]} v1.0.1 your_gitee_token")
        print(f"  {sys.argv[0]} v1.0.1 your_gitee_token your_github_token")
        print("\nGitee Token 获取方式:")
        print("  1. 命令行参数（第二个参数）")
        print("  2. 环境变量: export GITEE_TOKEN=***")
        print("  3. ~/.netrc 文件:")
        print("     machine gitee.com")
        print("     login your_username")
        print("     password your_token")
        print("\nGitHub Token（可选，用于双源发布）:")
        print("  环境变量: export GITHUB_TOKEN=***")
        print("  或: export HOMEBREW_TAP_TOKEN=***")
        print("\n获取 Token:")
        print("  Gitee: https://gitee.com/profile/personal_access_tokens")
        print("  GitHub: https://github.com/settings/tokens (Fine-grained, Contents R/W)")
        sys.exit(1)

    # 自动检测当前项目
    project_name, repo_path = detect_current_project()
    if not project_name or not repo_path:
        error("无法检测当前项目，请确保在项目目录下运行")
        sys.exit(1)

    info("=" * 60)
    info(f"项目: {project_name}")
    info(f"Gitee 仓库: {repo_path}")
    info(f"版本: {version}")
    if github_token:
        info(f"GitHub 双源发布: 启用（L00J/opsxcli + L00J/homebrew-tap）")
    else:
        info(f"GitHub 双源发布: 跳过（未提供 GITHUB_TOKEN）")
    info("=" * 60)
    info("")

    # ── 阶段 1: Gitee Release ──
    info("▸ 阶段 1/3: Gitee Release 上传")
    info("-" * 40)
    uploader = GiteeReleaseUploader(gitee_token)
    success, fail = uploader.process_project(".", repo_path, version)
    info("")
    if fail > 0:
        warn(f"Gitee 上传: 成功 {success} 个，失败 {fail} 个")
        warn("失败的文件已保留在 dist/ 目录，继续后续阶段...")
    else:
        info(f"✓ Gitee 上传完成: {success} 个文件")
        info(f"  查看: https://gitee.com/{repo_path}/releases")
    info("")

    # ── 阶段 2: GitHub Release（仅上传预编译二进制，不放源码）──
    github_success = 0
    github_fail = 0
    if github_token:
        info("▸ 阶段 2/3: GitHub Release 上传（预编译二进制）")
        info("-" * 40)
        gh_uploader = GitHubReleaseUploader(github_token)
        github_success, github_fail = gh_uploader.upload_dist_files(version)
        info("")
        if github_fail > 0:
            warn(f"GitHub 上传: 成功 {github_success} 个，失败 {github_fail} 个")
        else:
            info(f"✓ GitHub 上传完成: {github_success} 个文件")
            info(f"  查看: https://github.com/{GITHUB_REPO}/releases")
        info("")
    else:
        info("▸ 阶段 2/3: GitHub Release — 跳过（无 GITHUB_TOKEN）")
        info("")

    # ── 阶段 3: Homebrew Formula 推送到 L00J/homebrew-tap ──
    if github_token:
        info("▸ 阶段 3/3: Homebrew Formula 推送")
        info("-" * 40)
        formula_pusher = HomebrewFormulaPusher(github_token)
        ok = formula_pusher.push_formula(version)
        info("")
        if ok:
            info("✓ Homebrew Formula 已更新")
            info(f"  安装: brew tap L00J/tap && brew install opsxcli")
        else:
            warn("Homebrew Formula 推送失败（不影响已有发布）")
        info("")
    else:
        info("▸ 阶段 3/3: Homebrew Formula — 跳过（无 GITHUB_TOKEN）")
        info("")

    # ── 汇总 ──
    info("=" * 60)
    info("📋 发布汇总")
    info("=" * 60)
    info(f"  Gitee:   {success} 成功 / {fail} 失败")
    if github_token:
        info(f"  GitHub:  {github_success} 成功 / {github_fail} 失败")
        info(f"  Homebrew: {'✓ 已推送' if ok else '✗ 推送失败'}")
    else:
        info(f"  GitHub:  跳过")
        info(f"  Homebrew: 跳过")
    info("=" * 60)

    if fail > 0 or github_fail > 0:
        sys.exit(1)
    sys.exit(0)


if __name__ == "__main__":
    main()

