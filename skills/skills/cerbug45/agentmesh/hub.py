"""
AgentMesh – Hub (Message Broker)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
A Hub is the central router / directory-service for a group of agents.
Two implementations are provided:

  LocalHub   – in-process broker (perfect for single-process apps & testing)
  NetworkHub – TCP broker that spans processes / machines  (coming in v2)

The Hub never sees message *contents* – it only routes opaque encrypted
envelopes.  Agent discovery (public-key bundles) is the only plaintext
information stored.
"""

from __future__ import annotations

import json
import logging
import socket
import threading
from typing import Dict, List, Optional

logger = logging.getLogger("agentmesh.hub")


# ──────────────────────────────────────────────────────────────────────────────
# Base class / interface
# ──────────────────────────────────────────────────────────────────────────────

class Hub:
    """Abstract hub interface.  Subclass and implement the three methods below."""

    def register(self, agent) -> None:          # pragma: no cover
        raise NotImplementedError

    def get_bundle(self, agent_id: str) -> Optional[dict]:   # pragma: no cover
        raise NotImplementedError

    def deliver(self, envelope: dict) -> None:  # pragma: no cover
        raise NotImplementedError

    def list_agents(self) -> List[str]:         # pragma: no cover
        raise NotImplementedError


# ──────────────────────────────────────────────────────────────────────────────
# LocalHub  (in-process)
# ──────────────────────────────────────────────────────────────────────────────

class LocalHub(Hub):
    """
    In-process message hub.

    All agents registered here run in the *same* Python process.
    Messages are delivered synchronously in the caller's thread.
    Perfect for:
      • Unit tests
      • Multi-agent demos inside a single script
      • Jupyter notebooks

    Example
    -------
    >>> from agentmesh import Agent, LocalHub
    >>> hub = LocalHub()
    >>> alice = Agent("alice", hub=hub)
    >>> bob   = Agent("bob",   hub=hub)
    >>> @bob.on_message
    ... def echo(msg):
    ...     bob.send(msg.sender, text=f"Echo: {msg.text}")
    >>> alice.send("bob", text="ping")
    """

    def __init__(self):
        self._agents: Dict[str, object] = {}    # agent_id → Agent instance
        self._bundles: Dict[str, dict] = {}     # agent_id → public_bundle
        self._lock = threading.Lock()
        self._message_log: List[dict] = []      # optional audit log

    # ── Hub protocol ─────────────────────────────────────────────────────────

    def register(self, agent) -> None:
        with self._lock:
            self._agents[agent.id] = agent
            self._bundles[agent.id] = agent.public_bundle
        logger.debug("Registered agent %r", agent.id)

    def get_bundle(self, agent_id: str) -> Optional[dict]:
        return self._bundles.get(agent_id)

    def deliver(self, envelope: dict) -> None:
        recipient_id = envelope.get("to")
        with self._lock:
            recipient = self._agents.get(recipient_id)

        if recipient is None:
            logger.warning("Delivery failed: agent %r not found", recipient_id)
            return

        self._message_log.append({"ts": envelope.get("ts"), "from": envelope.get("from"), "to": recipient_id})
        recipient._receive(envelope)

    def list_agents(self) -> List[str]:
        with self._lock:
            return list(self._agents.keys())

    # ── Extras ───────────────────────────────────────────────────────────────

    def unregister(self, agent_id: str) -> None:
        with self._lock:
            self._agents.pop(agent_id, None)
            self._bundles.pop(agent_id, None)
        logger.debug("Unregistered agent %r", agent_id)

    def message_count(self) -> int:
        return len(self._message_log)

    def __repr__(self) -> str:
        return f"LocalHub(agents={list(self._agents.keys())})"


# ──────────────────────────────────────────────────────────────────────────────
# NetworkHub  (TCP, cross-process)
# ──────────────────────────────────────────────────────────────────────────────

class NetworkHub(Hub):
    """
    A lightweight TCP hub server.

    Start it in one process (or its own script):
    >>> hub_server = NetworkHubServer(host="0.0.0.0", port=7700)
    >>> hub_server.start()   # blocks, or run in a thread

    Agents in *any* process on the network connect via NetworkHub:
    >>> from agentmesh import Agent, NetworkHub
    >>> hub = NetworkHub(host="localhost", port=7700)
    >>> alice = Agent("alice", hub=hub)

    Protocol: newline-delimited JSON over TCP.
    Message types: REGISTER, GET_BUNDLE, DELIVER, LIST_AGENTS, BUNDLE_RESP, OK, ERROR
    """

    def __init__(self, host: str = "localhost", port: int = 7700):
        self.host = host
        self.port = port
        self._sock: Optional[socket.socket] = None
        self._lock = threading.Lock()

    def _connect(self) -> socket.socket:
        s = socket.create_connection((self.host, self.port), timeout=5)
        s.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
        return s

    def _send_recv(self, msg: dict) -> dict:
        with self._lock:
            if self._sock is None:
                self._sock = self._connect()
            try:
                self._sock.sendall((json.dumps(msg) + "\n").encode())
                data = b""
                while not data.endswith(b"\n"):
                    chunk = self._sock.recv(65536)
                    if not chunk:
                        raise ConnectionError("Hub closed connection")
                    data += chunk
                return json.loads(data.strip())
            except Exception:
                self._sock = None
                raise

    # ── Hub protocol ─────────────────────────────────────────────────────────

    def register(self, agent) -> None:
        resp = self._send_recv({"cmd": "REGISTER", "bundle": agent.public_bundle})
        if resp.get("status") != "OK":
            raise RuntimeError(f"Registration failed: {resp}")

    def get_bundle(self, agent_id: str) -> Optional[dict]:
        resp = self._send_recv({"cmd": "GET_BUNDLE", "agent_id": agent_id})
        return resp.get("bundle")

    def deliver(self, envelope: dict) -> None:
        resp = self._send_recv({"cmd": "DELIVER", "envelope": envelope})
        if resp.get("status") != "OK":
            logger.warning("Delivery error: %s", resp)

    def list_agents(self) -> List[str]:
        resp = self._send_recv({"cmd": "LIST_AGENTS"})
        return resp.get("agents", [])


# ──────────────────────────────────────────────────────────────────────────────
# NetworkHubServer  (the TCP server side)
# ──────────────────────────────────────────────────────────────────────────────

class NetworkHubServer:
    """
    Standalone TCP hub server.

    Agents connect via NetworkHub client.
    The server stores bundles and forwards envelopes to connected agents.

    Run as a module:
        python -m agentmesh.hub_server --host 0.0.0.0 --port 7700
    """

    def __init__(self, host: str = "0.0.0.0", port: int = 7700):
        self.host = host
        self.port = port
        self._bundles: Dict[str, dict] = {}
        self._agent_socks: Dict[str, socket.socket] = {}
        self._lock = threading.Lock()

    def start(self, block: bool = True) -> None:
        srv = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        srv.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        srv.bind((self.host, self.port))
        srv.listen(128)
        logger.info("NetworkHubServer listening on %s:%s", self.host, self.port)
        print(f"[AgentMesh Hub] Listening on {self.host}:{self.port}")

        if block:
            self._accept_loop(srv)
        else:
            t = threading.Thread(target=self._accept_loop, args=(srv,), daemon=True)
            t.start()

    def _accept_loop(self, srv: socket.socket) -> None:
        while True:
            conn, addr = srv.accept()
            t = threading.Thread(
                target=self._handle_client, args=(conn, addr), daemon=True
            )
            t.start()

    def _handle_client(self, conn: socket.socket, addr) -> None:
        registered_id: Optional[str] = None
        buf = b""
        try:
            while True:
                chunk = conn.recv(65536)
                if not chunk:
                    break
                buf += chunk
                while b"\n" in buf:
                    line, buf = buf.split(b"\n", 1)
                    msg = json.loads(line)
                    resp = self._handle_message(msg, conn)
                    if resp:
                        conn.sendall((json.dumps(resp) + "\n").encode())
                    if msg.get("cmd") == "REGISTER":
                        registered_id = msg.get("bundle", {}).get("agent_id")
                        with self._lock:
                            self._agent_socks[registered_id] = conn
        except Exception as exc:
            logger.debug("Client %s disconnected: %s", addr, exc)
        finally:
            if registered_id:
                with self._lock:
                    self._agent_socks.pop(registered_id, None)
            conn.close()

    def _handle_message(self, msg: dict, conn: socket.socket) -> Optional[dict]:
        cmd = msg.get("cmd")

        if cmd == "REGISTER":
            bundle = msg.get("bundle", {})
            agent_id = bundle.get("agent_id")
            with self._lock:
                self._bundles[agent_id] = bundle
            logger.debug("Registered %r", agent_id)
            return {"status": "OK"}

        elif cmd == "GET_BUNDLE":
            agent_id = msg.get("agent_id")
            bundle = self._bundles.get(agent_id)
            return {"bundle": bundle}

        elif cmd == "DELIVER":
            envelope = msg.get("envelope", {})
            recipient_id = envelope.get("to")
            with self._lock:
                recipient_sock = self._agent_socks.get(recipient_id)
            if recipient_sock:
                try:
                    push = {"cmd": "INCOMING", "envelope": envelope}
                    recipient_sock.sendall((json.dumps(push) + "\n").encode())
                except Exception as exc:
                    logger.warning("Push delivery failed: %s", exc)
                    return {"status": "ERROR", "reason": str(exc)}
            return {"status": "OK"}

        elif cmd == "LIST_AGENTS":
            with self._lock:
                agents = list(self._bundles.keys())
            return {"agents": agents}

        return {"status": "ERROR", "reason": f"Unknown command: {cmd}"}
