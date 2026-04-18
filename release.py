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
import requests
import subprocess
from pathlib import Path
from typing import Optional, Tuple, List

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
            if "linux-amd64" in filename:
                platforms["Linux x86_64"].append(filename)
            elif "linux-arm64" in filename:
                platforms["Linux ARM64"].append(filename)
            elif "darwin-amd64" in filename:
                platforms["macOS x86_64"].append(filename)
            elif "darwin-arm64" in filename:
                platforms["macOS ARM64"].append(filename)
            elif "windows-amd64" in filename:
                platforms["Windows x86_64"].append(filename)
            elif "windows-386" in filename:
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
            lines.append("chmod +x opsxcli")
            lines.append("sudo mv opsxcli /usr/local/bin/")
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


def get_token() -> Optional[str]:
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
    """主流程"""
    version = get_version()
    token = get_token()
    
    if not token:
        error("请提供 Gitee Token")
        print("\n使用方法:")
        print(f"  {sys.argv[0]} [版本号] [Gitee Token]")
        print("\n示例:")
        print(f"  {sys.argv[0]} v1.0.1")
        print(f"  {sys.argv[0]} v1.0.1 your_token")
        print("\nToken 获取方式:")
        print("  1. 命令行参数（第二个参数）")
        print("  2. 环境变量: export GITEE_TOKEN=your_token")
        print("  3. ~/.netrc 文件:")
        print("     machine gitee.com")
        print("     login your_username")
        print("     password your_token")
        print("\n获取 Token: https://gitee.com/profile/personal_access_tokens")
        sys.exit(1)
    
    # 自动检测当前项目
    project_name, repo_path = detect_current_project()
    if not project_name or not repo_path:
        error("无法检测当前项目，请确保在项目目录下运行")
        sys.exit(1)
    
    info("=" * 50)
    info(f"项目: {project_name}")
    info(f"仓库: {repo_path}")
    info(f"版本: {version}")
    info("=" * 50)
    info("")
    
    uploader = GiteeReleaseUploader(token)
    
    # 处理当前项目（使用当前目录作为项目目录）
    success, fail = uploader.process_project(".", repo_path, version)
    
    info("")
    info("=" * 50)
    if fail == 0:
        info(f"✓ 上传成功: {success} 个文件")
        info(f"查看 Release: https://gitee.com/{repo_path}/releases")
        sys.exit(0)
    else:
        warn(f"上传完成: 成功 {success} 个，失败 {fail} 个")
        warn("失败的文件已保留在 dist/ 目录")
        sys.exit(1)


if __name__ == "__main__":
    main()

