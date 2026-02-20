# MacPowerTools Skill

This skill provides a unified interface for### System Cleanup
- **Trash**: Empties the Trash.
- **Caches**: Cleans old cache files (older than 7 days) from `~/Library/Caches`.
- **Downloads**: Removes downloads older than 30 days.
- **Safety**: Defaults to `--dry-run`. Interactive confirmation required for deletion.

### Secure File Transfer (Android)
- **Android Integration**: Push files to Android devices via ADB.
- **Integrity Check**: Verifies SHA256 checksums on both ends.
- **Safety**:
    - Checks for existing files to prevent accidental overwrites.
    - Strict filename validation (alphanumeric/safe chars only).
    - Symlink detection to prevent transferring linked files.
    - Destination path restriction (must be `/sdcard/`, `/storage/`, or `/data/local/tmp`).

### External Drive Transfer
- **Auto-Detection**: Finds external drives (default keyword "Toshiba").
- **Write Check**: Detects if the drive is read-only (common with NTFS on macOS) and warns the user.
- **Smart Copy**: Handles both files and directories, merging folders if needed.
system maintenance and device interoperability on macOS.

## Installation

This skill is part of the `Tenqua/builtin_skills` package. Ensure dependencies are met:
- Python 3.6+
- `adb` (Android Debug Bridge) for transfer functionality.
    - Install via Homebrew: `brew install android-platform-tools`
- `transfer <source_file> [--dest <remote_path>] [--force]`: Securely transfer a file to a connected Android device.
- `drive-transfer <source> [--drive <keyword>] [--dest-folder <folder>] [--dry-run]`: Transfer data to an external drive.

## Architecture

The skill is modularized into:
- `scripts/power_tools.py`: Main CLI entry point.
- `scripts/cleanup.py`: Cleanup logic.
- `scripts/transfer.py`: Transfer logic.
- `scripts/utils.py`: Shared utilities.

## Usage Examples

```bash
# Clean up system (Dry Run)
cleanup

# Transfer file to Android
transfer ./release_build.apk --dest /sdcard/Builds/

# Transfer to Toshiba Drive
drive-transfer ./MyBackupFolder --drive Toshiba --dest-folder Backups
```

## Development

To add new tools, create a new module in `scripts/` and register it in `power_tools.py`.
