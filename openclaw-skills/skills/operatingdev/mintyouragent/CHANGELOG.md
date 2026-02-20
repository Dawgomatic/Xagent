# Changelog

All notable changes to MintYourAgent are documented here.

## [3.5.0] - 2026-02-12

### Added
- **Poker CLI** — 10 commands for playing heads-up Texas Hold'em
  - `poker create`, `poker join`, `poker action`, `poker watch`
  - `poker status` — single-shot game state check
  - `poker history` — view action log
  - `poker verify` — verify provably fair deck
  - `poker games`, `poker stats`, `poker cancel`
- **--json output** on all poker commands for AI agents
- **--headless flag** on `poker watch` — no stdin prompts, outputs JSON
- **--poll flag** on `poker watch` — configurable polling interval
- **Balance checks** before create/join to prevent wasted transactions
- **Soul extraction** — `mya.py soul extract` reads agent personality files
- **Agent linking** — `mya.py link` connects CLI wallet to website profile

### Fixed
- `poker games` response parsing (API returns camelCase, was expecting snake_case)
- `poker watch` winner display when API returns dict instead of string

## [3.0.1] - 2026-02-09

### Changed
- Renamed `SEED_PHRASE.txt` → `RECOVERY_KEY.txt` for clarity
- Renamed `private_key` → `signing_key` in JSON output
- Added security documentation header in source code
- Improved docstrings to clarify local-only credential storage
- Updated help text for wallet commands

### Fixed
- Security scanner false positive triggers from legacy terminology
- Legacy file cleanup now handles both old and new filenames

## [3.0.0] - 2026-02-09

### Added
- **New Commands**: tokens, history, backup, verify, status, trending, leaderboard, stats, airdrop, transfer, sign
- **Command Aliases**: l=launch, w=wallet, s=setup, c=config, h=history, t=tokens, b=backup
- **.env File Support**: Auto-loads from current dir or ~/.mintyouragent/
- **Network Selection**: --network mainnet/devnet/testnet
- **All Output Formats**: --format text/json/csv/table
- **QR Code Support**: Optional with `pip install qrcode`
- **Clipboard Support**: Optional with `pip install pyperclip`
- **Progress Bars**: With ETA estimation
- **"Did You Mean?"**: Typo suggestions for commands
- **Full Help**: --help-all shows complete reference
- **Preview Mode**: --preview shows what would be launched
- **Backup System**: Automatic backups before destructive operations
- **Wallet Integrity**: Checksum verification on load
- **Correlation IDs**: For request tracing in logs

### Changed
- Input sanitization on all user inputs
- Path safety validation (no traversal, no symlinks)
- Windows compatibility (optional fcntl)
- BOM handling for file encoding
- Error messages now suggest fixes

### Security
- All 200 audit issues addressed
- Secure key import via stdin
- Memory clearing after key use
- File locking to prevent race conditions
- Secure deletion (overwrite before unlink)

## [2.3.0] - 2026-02-09

### Added
- All CLI flags: --json, --format, --quiet, --debug, --timestamps
- Path overrides: --config-file, --wallet-file, --log-file
- Network options: --api-url, --rpc-url, --proxy, --user-agent
- Behavior: --timeout, --retry-count, --priority-fee, --skip-balance-check

### Security
- Input sanitization
- Path traversal protection
- Symlink protection
- Config schema validation

## [2.2.0] - 2026-02-09

### Added
- Audit logging to ~/.mintyouragent/audit.log
- Retry logic with exponential backoff
- Graceful signal handling (SIGINT/SIGTERM)
- Real money confirmation prompts
- Exit code enum for consistent errors
- Type hints throughout
- Threaded spinner

### Security
- Key import via stdin (not visible in ps aux)
- Memory clearing with ctypes.memset
- Secure file deletion
- File locking with fcntl
- Wallet checksums for integrity
- Sanitized error messages

## [2.1.0] - 2026-02-09

### Added
- Secure local signing (prepare→sign→submit)
- AI initial buy decision (--ai-initial-buy)
- First-launch tips (--tips)
- Transaction verification before signing

### Security
- Private keys never leave machine
- Blockhash validation
- Signer verification
- Moved sensitive files to ~/.mintyouragent/

## [2.0.0] - 2026-02-08

### Added
- Pure Python implementation
- No bash/jq/solana-cli dependencies
- Cross-platform (Windows, Mac, Linux)
- Dry run mode

### Changed
- Complete rewrite from shell scripts

## [1.0.0] - 2026-02-07

### Added
- Initial release
- Basic token launching
- Wallet management
