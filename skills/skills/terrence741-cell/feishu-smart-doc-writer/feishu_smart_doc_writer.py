#!/usr/bin/env python3
"""
Feishu Smart Doc Writer - 改进版
使用 OpenClaw 官方工具调用方式
"""

import re
import time
import asyncio
from typing import List, Optional
from dataclasses import dataclass

@dataclass
class ChunkConfig:
    """分块配置"""
    chunk_size: int = 2000          # 每块最大字符数
    max_retries: int = 3            # 最大重试次数
    retry_delay: float = 1.0        # 重试间隔（秒）
    show_progress: bool = True      # 显示进度
    convert_tables: bool = True     # 转换表格为文本


class ContentChunker:
    """内容分块器"""
    
    def __init__(self, config: ChunkConfig = None):
        self.config = config or ChunkConfig()
    
    def chunk_content(self, content: str) -> List[str]:
        """
        将长内容分割成多个小块
        策略：按段落分割，如果段落超过限制，按句子分割
        """
        chunks = []
        current_chunk = ""
        
        # 先处理表格
        if self.config.convert_tables:
            content = self._convert_tables(content)
        
        # 按段落分割
        paragraphs = self._split_paragraphs(content)
        
        for para in paragraphs:
            # 如果当前块加上新段落会超限
            if len(current_chunk) + len(para) > self.config.chunk_size:
                # 保存当前块
                if current_chunk.strip():
                    chunks.append(current_chunk.strip())
                
                # 如果单个段落就超限，需要进一步分割
                if len(para) > self.config.chunk_size:
                    sub_chunks = self._split_large_paragraph(para)
                    chunks.extend(sub_chunks)
                    current_chunk = ""
                else:
                    current_chunk = para
            else:
                current_chunk += "\n\n" + para if current_chunk else para
        
        # 保存最后一块
        if current_chunk.strip():
            chunks.append(current_chunk.strip())
        
        return chunks
    
    def _convert_tables(self, content: str) -> str:
        """将Markdown表格转换为文本列表"""
        table_pattern = r'\|[^\n]+\|\n\|[-:| ]+\|\n((?:\|[^\n]+\|\n)+)'
        
        def convert_table(match):
            table_text = match.group(0)
            lines = table_text.strip().split('\n')
            
            # 提取表头
            header = [cell.strip() for cell in lines[0].split('|')[1:-1]]
            
            # 提取数据行
            result = ["【表格内容】"]
            for line in lines[2:]:  # 跳过表头和分隔线
                cells = [cell.strip() for cell in line.split('|')[1:-1]]
                if cells and any(cells):  # 确保不是空行
                    row_text = ", ".join([f"{h}: {c}" for h, c in zip(header, cells)])
                    result.append(f"- {row_text}")
            
            return "\n".join(result)
        
        return re.sub(table_pattern, convert_table, content)
    
    def _split_paragraphs(self, content: str) -> List[str]:
        """按段落分割，保留标题结构"""
        lines = content.split('\n')
        paragraphs = []
        current_para = ""
        
        for line in lines:
            stripped = line.strip()
            
            if not stripped:
                if current_para.strip():
                    paragraphs.append(current_para.strip())
                    current_para = ""
                continue
            
            # 如果是标题，单独成段
            if stripped.startswith('#'):
                if current_para.strip():
                    paragraphs.append(current_para.strip())
                    current_para = ""
                paragraphs.append(stripped)
            else:
                current_para += line + "\n"
        
        if current_para.strip():
            paragraphs.append(current_para.strip())
        
        return paragraphs
    
    def _split_large_paragraph(self, para: str) -> List[str]:
        """分割大段落（按句子）"""
        chunks = []
        sentences = re.split(r'([。！？.\n])', para)
        current = ""
        
        for i in range(0, len(sentences), 2):
            sentence = sentences[i]
            if i + 1 < len(sentences):
                sentence += sentences[i + 1]  # 加上标点
            
            if len(current) + len(sentence) > self.config.chunk_size:
                if current.strip():
                    chunks.append(current.strip())
                current = sentence
            else:
                current += sentence
        
        if current.strip():
            chunks.append(current.strip())
        
        return chunks


class FeishuDocWriter:
    """
    飞书文档智能写入器
    使用 OpenClaw 官方工具调用方式
    """
    
    def __init__(self, ctx=None, config: ChunkConfig = None):
        """
        初始化
        
        Args:
            ctx: OpenClaw 上下文对象（在 Skill 中传入）
            config: 分块配置
        """
        self.ctx = ctx
        self.config = config or ChunkConfig()
        self.chunker = ContentChunker(config)
    
    async def write_document(self, title: str, content: str, folder_token: str = None) -> str:
        """
        创建新文档并写入内容（自动分块）
        
        Args:
            title: 文档标题
            content: 文档内容（支持长内容，自动分块）
            folder_token: 可选的文件夹token
            
        Returns:
            文档URL
        """
        if not self.ctx:
            raise ValueError("需要提供 OpenClaw 上下文对象 (ctx)")
        
        # 第一步：创建空文档（只传标题）
        doc_token = await self._create_empty_doc(title, folder_token)
        doc_url = f"https://feishu.cn/docx/{doc_token}"
        
        if self.config.show_progress:
            print(f" 文档创建成功: {doc_url}")
        
        # 第二步：分批追加内容
        success = await self._write_content_in_chunks(doc_token, content)
        
        if not success:
            raise Exception("写入内容失败")
        
        return doc_url
    
    async def append_to_document(self, doc_url: str, content: str) -> bool:
        """
        追加内容到现有文档（自动分块）
        
        Args:
            doc_url: 文档URL
            content: 要追加的内容
            
        Returns:
            是否成功
        """
        if not self.ctx:
            raise ValueError("需要提供 OpenClaw 上下文对象 (ctx)")
        
        doc_token = self._extract_token_from_url(doc_url)
        return await self._write_content_in_chunks(doc_token, content)
    
    async def _create_empty_doc(self, title: str, folder_token: str = None) -> str:
        """创建空文档，只传标题"""
        try:
            # 使用 OpenClaw 官方工具调用方式
            result = await self.ctx.invoke_tool("feishu_doc.create", {
                "title": title,
                "folder_token": folder_token
            })
            
            # 提取 doc_token
            if isinstance(result, dict):
                doc_token = result.get("document_id") or result.get("doc_token")
                if doc_token:
                    return doc_token
            
            # 如果是字符串，尝试提取
            if isinstance(result, str):
                import re
                match = re.search(r'docx/([a-zA-Z0-9]+)', result)
                if match:
                    return match.group(1)
                return result
            
            raise Exception(f"无法解析文档token: {result}")
            
        except Exception as e:
            raise Exception(f"创建文档失败: {e}")
    
    async def _write_content_in_chunks(self, doc_token: str, content: str) -> bool:
        """分批写入内容"""
        chunks = self.chunker.chunk_content(content)
        
        if self.config.show_progress:
            print(f" 内容已分割为 {len(chunks)} 块，开始写入...")
        
        for i, chunk in enumerate(chunks, 1):
            if self.config.show_progress:
                print(f"  写入第 {i}/{len(chunks)} 块 ({len(chunk)} 字符)...")
            
            success = await self._append_chunk_with_retry(doc_token, chunk)
            
            if not success:
                print(f" 第 {i} 块写入失败")
                return False
            
            # 添加小延迟，避免API限流
            if i < len(chunks):
                await asyncio.sleep(0.5)
        
        if self.config.show_progress:
            print(f" 全部 {len(chunks)} 块写入完成")
        
        return True
    
    async def _append_chunk_with_retry(self, doc_token: str, chunk: str) -> bool:
        """带重试的追加内容"""
        for attempt in range(self.config.max_retries):
            try:
                return await self._append_chunk(doc_token, chunk)
            except Exception as e:
                if self.config.show_progress:
                    print(f"    尝试 {attempt + 1}/{self.config.max_retries} 失败: {e}")
                if attempt < self.config.max_retries - 1:
                    await asyncio.sleep(self.config.retry_delay * (attempt + 1))
                else:
                    return False
        return False
    
    async def _append_chunk(self, doc_token: str, chunk: str) -> bool:
        """追加单块内容"""
        try:
            # 使用 OpenClaw 官方工具调用方式
            await self.ctx.invoke_tool("feishu_doc.append", {
                "doc_token": doc_token,
                "content": chunk
            })
            return True
        except Exception as e:
            raise Exception(f"API调用失败: {e}")
    
    def _extract_token_from_url(self, url: str) -> str:
        """从URL中提取doc_token"""
        import re
        match = re.search(r'docx/([a-zA-Z0-9]+)', url)
        if match:
            return match.group(1)
        raise ValueError(f"无法从URL提取token: {url}")
    
    async def transfer_ownership(self, doc_url: str, owner_openid: str) -> bool:
        """
        转移文档所有权
        
        Args:
            doc_url: 文档URL
            owner_openid: 新所有者的openid (例如: ou_xxxxxxxx)
        
        Returns:
            是否成功
        """
        if not self.ctx:
            raise ValueError("需要提供 OpenClaw 上下文对象 (ctx)")
        
        doc_token = self._extract_token_from_url(doc_url)
        
        try:
            # 使用 OpenClaw 工具调用转移所有权
            await self.ctx.invoke_tool("feishu_doc.transfer_owner", {
                "doc_token": doc_token,
                "member_type": "openid",
                "member_id": owner_openid
            })
            
            if self.config.show_progress:
                print(f" 文档所有权已转移给 {owner_openid}")
            
            return True
            
        except Exception as e:
            if self.config.show_progress:
                print(f" 所有权转移失败: {e}")
            
            # 转移失败不影响文档创建，记录错误但不抛出
            return False
    
    async def write_document_with_transfer(
        self, 
        title: str, 
        content: str, 
        folder_token: str = None,
        owner_openid: str = None
    ) -> dict:
        """
        创建文档并写入内容，完成后自动转移所有权
        
        Args:
            title: 文档标题
            content: 文档内容
            folder_token: 可选的文件夹token
            owner_openid: 新所有者的openid，如果提供则自动转移所有权
        
        Returns:
            {
                "doc_url": "...",
                "doc_token": "...",
                "chunks_count": N,
                "owner_transferred": True/False
            }
        """
        # 1. 创建并写入文档
        doc_url = await self.write_document(title, content, folder_token)
        
        # 2. 提取 doc_token
        doc_token = self._extract_token_from_url(doc_url)
        
        # 3. 计算分块数
        chunks = self.chunker.chunk_content(content)
        
        # 4. 转移所有权（如果提供了 owner_openid）
        owner_transferred = False
        if owner_openid:
            owner_transferred = await self.transfer_ownership(doc_url, owner_openid)
        
        return {
            "doc_url": doc_url,
            "doc_token": doc_token,
            "chunks_count": len(chunks),
            "owner_transferred": owner_transferred
        }


# 同步包装函数（方便非异步环境使用）
def write_document_sync(ctx, title: str, content: str, folder_token: str = None, config: ChunkConfig = None) -> str:
    """同步方式写入文档"""
    writer = FeishuDocWriter(ctx, config)
    return asyncio.run(writer.write_document(title, content, folder_token))


def write_document_with_transfer_sync(
    ctx, 
    title: str, 
    content: str, 
    folder_token: str = None,
    owner_openid: str = None,
    config: ChunkConfig = None
) -> dict:
    """同步方式写入文档并转移所有权"""
    writer = FeishuDocWriter(ctx, config)
    return asyncio.run(writer.write_document_with_transfer(title, content, folder_token, owner_openid))


def append_to_document_sync(ctx, doc_url: str, content: str, config: ChunkConfig = None) -> bool:
    """同步方式追加文档"""
    writer = FeishuDocWriter(ctx, config)
    return asyncio.run(writer.append_to_document(doc_url, content))
