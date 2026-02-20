"""
AgentMesh – Agent
━━━━━━━━━━━━━━━━━
A self-contained AI agent identity that can:
  • Register to a Hub (broker) or run peer-to-peer
  • Establish encrypted sessions with other agents
  • Send / receive signed + encrypted messages
  • Auto-discover peers via the Hub
  • Persist its key-pair across restarts
"""

from __future__ import annotations

import json
import time
import threading
import logging
from pathlib import Path
from typing import Callable, Dict, List, Optional, Any

from .crypto import (
    AgentKeyPair,
    CryptoSession,
    CryptoError,
    perform_key_exchange,
    seal,
    unseal,
)
from .transport import Transport, LocalTransport

logger = logging.getLogger("agentmesh.agent")


# ──────────────────────────────────────────────────────────────────────────────
# Message model
# ──────────────────────────────────────────────────────────────────────────────

class Message:
    """A decrypted, verified message received from a peer agent."""

    __slots__ = ("sender", "recipient", "payload", "timestamp", "raw_envelope")

    def __init__(
        self,
        sender: str,
        recipient: str,
        payload: dict,
        timestamp: int,
        raw_envelope: dict,
    ):
        self.sender = sender
        self.recipient = recipient
        self.payload = payload
        self.timestamp = timestamp
        self.raw_envelope = raw_envelope

    def __repr__(self) -> str:
        return (
            f"Message(from={self.sender!r}, to={self.recipient!r}, "
            f"payload={self.payload!r})"
        )

    @property
    def text(self) -> Optional[str]:
        """Convenience accessor for 'text' key in payload."""
        return self.payload.get("text")

    @property
    def type(self) -> str:
        """Convenience accessor for 'type' key in payload (default: 'message')."""
        return self.payload.get("type", "message")


# ──────────────────────────────────────────────────────────────────────────────
# Agent
# ──────────────────────────────────────────────────────────────────────────────

class Agent:
    """
    An AI agent with a cryptographic identity.

    Quick-start
    -----------
    >>> from agentmesh import Agent, LocalHub
    >>> hub = LocalHub()
    >>> alice = Agent("alice", hub=hub)
    >>> bob   = Agent("bob",   hub=hub)
    >>> alice.on_message(lambda msg: print(f"Bob says: {msg.text}"))
    >>> bob.send("alice", text="Hello from Bob!")

    Parameters
    ----------
    agent_id : str
        Human-readable identifier for this agent.  Must be unique on the hub.
    hub : Hub-like object, optional
        A Hub instance (LocalHub or NetworkHub).  Required for peer discovery.
    keypair_path : str | Path, optional
        Path to a JSON file for persisting the key-pair.
        If the file exists the keys are loaded; otherwise new keys are generated
        and saved there.
    log_level : int, optional
        Logging level for this agent.  Default: logging.WARNING.
    """

    def __init__(
        self,
        agent_id: str,
        hub=None,
        keypair_path: Optional[str | Path] = None,
        log_level: int = logging.WARNING,
    ):
        self.id = agent_id
        self._hub = hub
        self._sessions: Dict[str, CryptoSession] = {}
        self._peer_bundles: Dict[str, dict] = {}
        self._handlers: List[Callable[[Message], Any]] = []
        self._lock = threading.Lock()

        logging.getLogger("agentmesh").setLevel(log_level)

        # Load or generate key-pair
        if keypair_path:
            self._keypair = _load_or_create_keypair(Path(keypair_path))
        else:
            self._keypair = AgentKeyPair()

        # Register with hub
        if self._hub is not None:
            self._hub.register(self)

        logger.info(
            "Agent %r ready. Fingerprint: %s", self.id, self._keypair.fingerprint
        )

    # ── Identity ──────────────────────────────────────────────────────────────

    @property
    def public_bundle(self) -> dict:
        """Public key bundle to share with peers."""
        bundle = self._keypair.public_bundle()
        bundle["agent_id"] = self.id
        return bundle

    @property
    def fingerprint(self) -> str:
        """Hex fingerprint of this agent's identity key."""
        return self._keypair.fingerprint

    # ── Session management ────────────────────────────────────────────────────

    def connect(self, peer_id: str) -> None:
        """
        Establish (or refresh) an encrypted session with *peer_id*.
        Fetches the peer's public bundle from the hub and performs ECDH.
        """
        if self._hub is None:
            raise RuntimeError("No hub configured – cannot auto-discover peers")

        bundle = self._hub.get_bundle(peer_id)
        if bundle is None:
            raise ValueError(f"Peer {peer_id!r} not found on hub")

        with self._lock:
            self._peer_bundles[peer_id] = bundle
            self._sessions[peer_id] = perform_key_exchange(self._keypair, bundle)
        logger.debug("Session established: %s ↔ %s", self.id, peer_id)

    def connect_with_bundle(self, bundle: dict) -> None:
        """
        Establish a session given an already-obtained public bundle.
        Useful for peer-to-peer mode (no hub).
        """
        peer_id = bundle["agent_id"]
        with self._lock:
            self._peer_bundles[peer_id] = bundle
            self._sessions[peer_id] = perform_key_exchange(self._keypair, bundle)
        logger.debug("Session (bundle) established: %s ↔ %s", self.id, peer_id)

    def _ensure_session(self, peer_id: str) -> CryptoSession:
        if peer_id not in self._sessions:
            self.connect(peer_id)
        return self._sessions[peer_id]

    # ── Sending ───────────────────────────────────────────────────────────────

    def send(self, recipient_id: str, *, text: str = "", **extra) -> None:
        """
        Send an encrypted message to *recipient_id*.

        The message payload is a dict; you can pass arbitrary keyword args
        which will be merged in.  The special kwarg ``text=`` sets the
        human-readable body.

        Example
        -------
        >>> alice.send("bob", text="Hello!", priority="high")
        """
        payload: dict = {"type": "message", "text": text}
        payload.update(extra)
        self.send_payload(recipient_id, payload)

    def send_payload(self, recipient_id: str, payload: dict) -> None:
        """
        Low-level send: encrypt *payload* and deliver to *recipient_id*.
        """
        session = self._ensure_session(recipient_id)
        envelope = seal(session, self._keypair, self.id, recipient_id, payload)

        if self._hub is not None:
            self._hub.deliver(envelope)
        else:
            raise RuntimeError(
                "No hub configured.  For P2P use, call "
                "agent.send_envelope(recipient_agent, envelope) directly."
            )

    def send_envelope(self, recipient_agent: "Agent", envelope: dict) -> None:
        """
        Direct (P2P, no hub) delivery.  Calls recipient._receive() directly.
        Useful for testing or in-process setups.
        """
        recipient_agent._receive(envelope)

    # ── Receiving ─────────────────────────────────────────────────────────────

    def _receive(self, envelope: dict) -> None:
        """
        Internal: decrypt and dispatch an incoming envelope.
        Called by the hub or directly by a peer in P2P mode.
        """
        sender_id = envelope.get("from", "")
        try:
            sender_bundle = self._peer_bundles.get(sender_id)
            if sender_bundle is None:
                # Auto-fetch sender's bundle on first message
                if self._hub is not None:
                    sender_bundle = self._hub.get_bundle(sender_id)
                    if sender_bundle:
                        with self._lock:
                            self._peer_bundles[sender_id] = sender_bundle
                            self._sessions[sender_id] = perform_key_exchange(
                                self._keypair, sender_bundle
                            )
                if sender_bundle is None:
                    raise CryptoError(f"Unknown sender: {sender_id!r}")

            session = self._sessions[sender_id]
            payload = unseal(session, envelope, sender_bundle)

            msg = Message(
                sender=sender_id,
                recipient=envelope.get("to", self.id),
                payload=payload,
                timestamp=envelope.get("ts", 0),
                raw_envelope=envelope,
            )
            self._dispatch(msg)

        except CryptoError as exc:
            logger.warning("CryptoError from %r: %s", sender_id, exc)
        except Exception as exc:
            logger.error("Unexpected error receiving from %r: %s", sender_id, exc)

    def _dispatch(self, msg: Message) -> None:
        for handler in self._handlers:
            try:
                handler(msg)
            except Exception as exc:
                logger.error("Handler error: %s", exc)

    # ── Handler registration ───────────────────────────────────────────────────

    def on_message(self, handler: Callable[[Message], Any]) -> "Agent":
        """
        Register a callback for incoming messages.

        Can be used as a decorator:
        >>> @alice.on_message
        ... def handle(msg):
        ...     print(msg.text)

        Or called directly:
        >>> alice.on_message(lambda msg: print(msg.text))

        Returns self for chaining.
        """
        self._handlers.append(handler)
        return self

    # ── Discovery ────────────────────────────────────────────────────────────

    def list_peers(self) -> List[str]:
        """Return IDs of all agents registered on the hub."""
        if self._hub is None:
            raise RuntimeError("No hub configured")
        return [p for p in self._hub.list_agents() if p != self.id]

    # ── Diagnostics ──────────────────────────────────────────────────────────

    def status(self) -> dict:
        return {
            "agent_id": self.id,
            "fingerprint": self.fingerprint,
            "active_sessions": list(self._sessions.keys()),
            "known_peers": list(self._peer_bundles.keys()),
            "handlers": len(self._handlers),
        }

    def __repr__(self) -> str:
        return f"Agent(id={self.id!r}, fingerprint={self.fingerprint[:19]}…)"


# ──────────────────────────────────────────────────────────────────────────────
# Key persistence
# ──────────────────────────────────────────────────────────────────────────────

def _load_or_create_keypair(path: Path) -> AgentKeyPair:
    if path.exists():
        with path.open() as fh:
            data = json.load(fh)
        logger.info("Loaded key-pair from %s", path)
        return AgentKeyPair.from_dict(data)

    kp = AgentKeyPair()
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w") as fh:
        json.dump(kp.to_dict(), fh, indent=2)
    logger.info("Generated new key-pair, saved to %s", path)
    return kp
