# Runtime Security Checks

## Data Source

Run `scripts/runtime_check.sh` to gather live system state.

## Check 1: Network Exposure

**From**: `runtime.network`

### Direct Internet Exposure

| Condition | Risk |
|-----------|------|
| `potentially_exposed: true` + `behind_nat: false` |  Critical |
| `potentially_exposed: true` + `behind_nat: true` |  Medium (NAT helps) |
| `potentially_exposed: false` |  Low |

**Critical Finding** (`exposed + no NAT`):
```
Gateway is binding to all interfaces (0.0.0.0) and your external IP 
matches a local IP, meaning you may be directly exposed to the internet 
without NAT protection.

Immediate actions:
1. Check if port {gateway_port} is reachable from outside
2. Enable firewall: sudo ufw allow from <trusted_ip> to any port {gateway_port}
3. Or change gateway.bind to "localhost" or specific LAN IP
```

### VPN Status

| Condition | Impact |
|-----------|--------|
| `vpn_type: wireguard/tailscale` | Reduces network exposure severity |
| `vpn_type: none` + LAN bind |  Medium (depends on network trust) |
| `vpn_type: none` + 0.0.0.0 bind |  High |

## Check 2: Container/VM Isolation

**From**: `runtime.isolation`

### Bare Metal vs Container

| Environment | Security Implication |
|-------------|---------------------|
| `in_container: true` |  Some isolation from host |
| `in_container: false` + `in_vm: true` |  VM-level isolation |
| `in_container: false` + `in_vm: false` |  Bare metal - full system access |

**Bare Metal Finding**:
```
OpenClaw is running directly on the host system without container 
or VM isolation. If compromised, an attacker has full access to:
- All files on the system
- Network interfaces
- Other running processes
- Potentially other users' data

Consider:
- Running in Docker for isolation
- Using a dedicated VM
- At minimum, running as non-root user with limited sudo
```

## Check 3: Privilege Level

**From**: `runtime.privileges`

### Root Access

| Condition | Risk |
|-----------|------|
| `running_as_root: true` |  Critical |
| `can_sudo: true` (passwordless) |  High |
| `can_sudo: true` (with password) |  Medium |
| `can_sudo: false` |  Low |

**Running as Root Finding**:
```
 CRITICAL: OpenClaw is running as root!

This means any prompt injection that triggers exec can:
- Modify any file on the system
- Install rootkits/backdoors
- Access all users' data
- Modify system configuration

Immediate action: Run OpenClaw as a non-root user.
```

**Passwordless Sudo Finding**:
```
 HIGH: OpenClaw user can run sudo without password.

An attacker who achieves command execution can escalate to root.
Consider restricting sudo access or requiring password.
```

### Capabilities Check

If `capabilities` is not "none":
- Check which capabilities are granted
- `CAP_NET_ADMIN`, `CAP_SYS_ADMIN` =  High risk
- `CAP_NET_BIND_SERVICE` only =  Low (common for ports <1024)

## Check 4: File Permissions

**From**: `runtime.filesystem`

### OpenClaw Directory

| Condition | Risk |
|-----------|------|
| `openclaw_dir_perms` not 700 |  Medium |
| `config_perms` not 600 |  Medium |
| `credentials_dir_perms` not 700 |  High |
| `world_readable_sensitive_files > 0` |  High |

**Permissions Finding**:
```
Sensitive files are readable by other users on this system.

Files in ~/.openclaw may contain:
- API tokens (Anthropic, OpenAI)
- Channel tokens (Slack, Telegram, Discord)
- Gateway authentication credentials

Fix:
chmod 700 ~/.openclaw
chmod 600 ~/.openclaw/openclaw.json
chmod -R 700 ~/.openclaw/credentials
```

## Check 5: Resource Limits

**From**: `runtime.resources`

Informational checks for DoS resilience:

| Resource | Concern |
|----------|---------|
| `max_open_files` < 1024 | May limit concurrent connections |
| `max_processes` < 100 | May limit subagent spawning |
| `disk_free_mb` < 1000 | Session logs may fill disk |

These are  Info level unless extremely constrained.

## Risk Summary Matrix

```
                    Isolation Level
                 Container    Bare Metal
            ┌────────────┬────────────┐
Privileges  │            │            │
   Root     │   High   │  Critical│
            ├────────────┼────────────┤
   Sudo     │   Medium │   High   │
            ├────────────┼────────────┤
   Limited  │   Low    │   Medium │
            └────────────┴────────────┘
```

## Combined Assessment

Generate an overall runtime security score:

1. **Network Exposure**: Exposed + No VPN + No NAT = 
2. **Privilege Level**: Root or passwordless sudo = /
3. **Isolation**: Bare metal + root = 
4. **File Permissions**: World-readable creds = 

If any : Overall =  Critical
If any  and no : Overall =  High
If only : Overall =  Medium
Otherwise:  Good
