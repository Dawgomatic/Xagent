# ipfs-client

Read-only IPFS queries — fetch files, inspect metadata, explore DAGs, and resolve IPNS names via local or public gateways.

## Quick Start

```bash
# Set up free public gateway
export IPFS_GATEWAY="https://ipfs.io"

# Fetch a file (no setup needed)
curl "$IPFS_GATEWAY/ipfs/QmYwAPJzv5CZsnAzt8auVZcgSDUbkXz2x4k2k5xmj8W1gR"

# Resolve IPNS name
curl "$IPFS_GATEWAY/ipns/docs.ipfs.tech"
```

## Key Features

- ** Read-only:** No publishing, no pinning, completely safe exploration
- ** Public gateways:** Instant access via ipfs.io, cloudflare-ipfs.com, dweb.link
- ** Content inspection:** DAG exploration, metadata analysis, file type detection
- ** IPNS resolution:** Resolve mutable names and ENS+IPFS integration

## Requirements

- `ipfs` CLI (optional) or `curl` for gateway requests
- No IPFS node required - works with public gateways

See [SKILL.md](./SKILL.md) for complete documentation.