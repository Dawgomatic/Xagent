#!/usr/bin/env python3
"""
memory_bridge.py - SWE100821
Bridge between PicoClaw Markdown memory and Qdrant vector database.
Enables semantic search over long-term agent memory using local embeddings.
"""

import os
import sys
import time
import requests
from pathlib import Path
from datetime import datetime
from typing import List, Dict, Optional

try:
    from qdrant_client import QdrantClient
    from qdrant_client.models import Distance, VectorParams, PointStruct
except ImportError:
    print("ERROR: qdrant-client not installed. Run: pip install qdrant-client")
    sys.exit(1)

try:
    from watchdog.observers import Observer
    from watchdog.events import FileSystemEventHandler
    WATCHDOG_AVAILABLE = True
except ImportError:
    WATCHDOG_AVAILABLE = False


class DistributedMemory:
    """Vector-based long-term memory for distributed agent system."""
    
    def __init__(
        self, 
        qdrant_url: str = "http://localhost:6333",
        ollama_url: str = "http://localhost:11434",
        collection: str = "agent_memory"
    ):
        """Initialize connection to Qdrant and Ollama.
        
        Args:
            qdrant_url: Qdrant server URL
            ollama_url: Ollama server URL
            collection: Qdrant collection name
        """
        self.qdrant_url = qdrant_url
        self.ollama_url = ollama_url
        self.collection = collection
        
        try:
            self.qdrant = QdrantClient(url=qdrant_url)
        except Exception as e:
            print(f"ERROR: Cannot connect to Qdrant at {qdrant_url}")
            print(f"Make sure Qdrant is running: docker run -d -p 6333:6333 qdrant/qdrant")
            sys.exit(1)
        
        # Create collection if doesn't exist
        self._ensure_collection()
    
    def _ensure_collection(self):
        """Create collection if it doesn't exist."""
        try:
            self.qdrant.get_collection(self.collection)
            print(f"✓ Connected to collection: {self.collection}")
        except:
            print(f"Creating new collection: {self.collection}")
            self.qdrant.create_collection(
                collection_name=self.collection,
                vectors_config=VectorParams(
                    size=768,  # nomic-embed-text dimension
                    distance=Distance.COSINE
                )
            )
            print(f"✓ Collection created: {self.collection}")
    
    def get_embedding(self, text: str) -> List[float]:
        """Get embedding from local Ollama.
        
        Args:
            text: Text to embed
            
        Returns:
            768-dimensional embedding vector
        """
        try:
            resp = requests.post(
                f"{self.ollama_url}/api/embeddings",
                json={"model": "nomic-embed-text", "prompt": text},
                timeout=30
            )
            resp.raise_for_status()
            return resp.json()["embedding"]
        except Exception as e:
            print(f"ERROR: Cannot get embedding from Ollama: {e}")
            print(f"Make sure nomic-embed-text is installed: ollama pull nomic-embed-text")
            sys.exit(1)
    
    def add_memory(self, text: str, metadata: Optional[Dict] = None) -> int:
        """Add memory to vector DB.
        
        Args:
            text: Memory text
            metadata: Additional metadata (optional)
            
        Returns:
            Point ID
        """
        embedding = self.get_embedding(text)
        
        if metadata is None:
            metadata = {}
        
        # Add standard metadata
        metadata.update({
            "text": text,
            "timestamp": datetime.now().isoformat(),
            "node": os.uname().nodename
        })
        
        # Generate unique ID
        point_id = abs(hash(text + str(datetime.now()))) % (10 ** 10)
        
        self.qdrant.upsert(
            collection_name=self.collection,
            points=[PointStruct(
                id=point_id,
                vector=embedding,
                payload=metadata
            )]
        )
        
        return point_id
    
    def search_memory(self, query: str, limit: int = 5) -> List[Dict]:
        """Search memories by semantic similarity.
        
        Args:
            query: Search query
            limit: Maximum results
            
        Returns:
            List of matching memories with scores
        """
        query_embedding = self.get_embedding(query)
        
        results = self.qdrant.search(
            collection_name=self.collection,
            query_vector=query_embedding,
            limit=limit
        )
        
        return [
            {
                "text": hit.payload["text"],
                "score": hit.score,
                "timestamp": hit.payload.get("timestamp", "unknown"),
                "node": hit.payload.get("node", "unknown"),
                "source": hit.payload.get("source", "manual")
            }
            for hit in results
        ]
    
    def sync_from_markdown(self, memory_file: Path) -> int:
        """Import existing MEMORY.md into vector DB.
        
        Args:
            memory_file: Path to MEMORY.md
            
        Returns:
            Number of entries synced
        """
        if not memory_file.exists():
            print(f"WARNING: {memory_file} not found")
            return 0
        
        print(f"Reading {memory_file}...")
        content = memory_file.read_text(encoding="utf-8")
        
        # Split by headers or double newlines (paragraphs)
        sections = []
        
        # Try splitting by headers first
        lines = content.split("\n")
        current_section = []
        
        for line in lines:
            if line.startswith("#") and current_section:
                # Save previous section
                section_text = "\n".join(current_section).strip()
                if len(section_text) > 20:  # Skip very short entries
                    sections.append(section_text)
                current_section = [line]
            else:
                current_section.append(line)
        
        # Add last section
        if current_section:
            section_text = "\n".join(current_section).strip()
            if len(section_text) > 20:
                sections.append(section_text)
        
        # If no headers, split by paragraphs
        if len(sections) < 2:
            sections = [s.strip() for s in content.split("\n\n") if len(s.strip()) > 20]
        
        # Add each section to vector DB
        synced = 0
        for section in sections:
            try:
                self.add_memory(section, metadata={"source": str(memory_file.name)})
                synced += 1
                print(f"  ✓ Synced: {section[:60]}...")
            except Exception as e:
                print(f"  ✗ Failed to sync section: {e}")
        
        print(f"\n✅ Synced {synced} entries from {memory_file}")
        return synced


class MemoryWatcher(FileSystemEventHandler):
    """Watch MEMORY.md for changes and auto-sync to vector DB."""
    
    def __init__(self, memory_file: Path, distributed_mem: DistributedMemory):
        self.memory_file = memory_file
        self.distributed_mem = distributed_mem
        self.last_sync = 0
    
    def on_modified(self, event):
        """Handle file modification events."""
        if event.src_path == str(self.memory_file):
            # Debounce (wait 5 seconds after last change)
            current_time = time.time()
            if current_time - self.last_sync > 5:
                print(f"\n📝 Detected change in {self.memory_file}, syncing...")
                try:
                    self.distributed_mem.sync_from_markdown(self.memory_file)
                    self.last_sync = current_time
                except Exception as e:
                    print(f"✗ Sync failed: {e}")


def watch_mode(memory_file: Path):
    """Watch MEMORY.md and auto-sync to vector DB.
    
    Args:
        memory_file: Path to MEMORY.md to watch
    """
    if not WATCHDOG_AVAILABLE:
        print("ERROR: watchdog not installed. Run: pip install watchdog")
        sys.exit(1)
    
    if not memory_file.exists():
        print(f"WARNING: {memory_file} does not exist yet, creating...")
        memory_file.parent.mkdir(parents=True, exist_ok=True)
        memory_file.write_text("# Long-term Memory\n\n(No entries yet)\n")
    
    mem = DistributedMemory()
    
    # Initial sync
    print("Initial sync...")
    mem.sync_from_markdown(memory_file)
    
    # Start watching
    event_handler = MemoryWatcher(memory_file, mem)
    observer = Observer()
    observer.schedule(event_handler, str(memory_file.parent), recursive=False)
    observer.start()
    
    print(f"\n👀 Watching {memory_file} for changes...")
    print("Press Ctrl+C to stop")
    
    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        print("\nStopping watcher...")
        observer.stop()
    observer.join()


def main():
    """Main CLI entry point."""
    if len(sys.argv) < 2:
        print("""
memory_bridge.py - SWE100821
Bridge between Markdown memory and Qdrant vector database

Usage:
  python memory_bridge.py sync              # Import MEMORY.md to vector DB
  python memory_bridge.py search <query>    # Search memories semantically
  python memory_bridge.py add <text>        # Add memory entry
  python memory_bridge.py watch             # Watch MEMORY.md and auto-sync

Examples:
  python memory_bridge.py sync
  python memory_bridge.py search "Rust web frameworks"
  python memory_bridge.py add "Decided to use Axum for production API"
  python memory_bridge.py watch  # Run as service
""")
        sys.exit(1)
    
    command = sys.argv[1]
    mem = DistributedMemory()
    
    if command == "sync":
        # Sync from MEMORY.md
        memory_file = Path.home() / ".picoclaw" / "workspace" / "memory" / "MEMORY.md"
        
        if len(sys.argv) > 2:
            # Custom path provided
            memory_file = Path(sys.argv[2])
        
        mem.sync_from_markdown(memory_file)
    
    elif command == "search":
        # Search memories
        if len(sys.argv) < 3:
            print("ERROR: Please provide a search query")
            sys.exit(1)
        
        query = " ".join(sys.argv[2:])
        print(f"\n🔍 Search: {query}\n")
        
        results = mem.search_memory(query, limit=10)
        
        if not results:
            print("No results found.")
            sys.exit(0)
        
        for i, result in enumerate(results, 1):
            score_bar = "█" * int(result['score'] * 10)
            print(f"{i}. [{score_bar} {result['score']:.3f}]")
            print(f"   {result['text'][:200]}...")
            print(f"   📍 {result['node']} | 🕐 {result['timestamp'][:19]} | 📄 {result['source']}")
            print()
    
    elif command == "add":
        # Add memory
        if len(sys.argv) < 3:
            print("ERROR: Please provide text to remember")
            sys.exit(1)
        
        text = " ".join(sys.argv[2:])
        point_id = mem.add_memory(text)
        print(f"✅ Memory added (ID: {point_id}): {text[:60]}...")
    
    elif command == "watch":
        # Watch mode
        memory_file = Path.home() / ".picoclaw" / "workspace" / "memory" / "MEMORY.md"
        
        if len(sys.argv) > 2:
            memory_file = Path(sys.argv[2])
        
        watch_mode(memory_file)
    
    else:
        print(f"ERROR: Unknown command: {command}")
        print("Run without arguments to see usage")
        sys.exit(1)


if __name__ == "__main__":
    main()
