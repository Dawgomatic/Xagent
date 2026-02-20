#!/usr/bin/env python3
import argparse
import sys
import os

# Ensure the skill root directory is in the python path so that 'scripts' module can be imported
current_dir = os.path.dirname(os.path.abspath(__file__))
skill_root = os.path.dirname(current_dir)
if skill_root not in sys.path:
    sys.path.insert(0, skill_root)

from scripts.cleanup import run_cleanup
from scripts.transfer import run_transfer
from scripts.drive_transfer import run_drive_transfer

def main():
    parser = argparse.ArgumentParser(description="MacPowerTools: A collection of utilities for macOS power users.")
    subparsers = parser.add_subparsers(dest="command", help="Available commands")

    # Subcommand: cleanup
    parser_cleanup = subparsers.add_parser("cleanup", help="Clean up system caches, trash, and old downloads.")
    parser_cleanup.add_argument("--dry-run", action="store_true", help="Print actions without deleting files", default=True)
    parser_cleanup.add_argument("--force", action="store_false", dest="dry_run", help="Actually delete files")

    # Subcommand: transfer (Android)
    parser_transfer = subparsers.add_parser("transfer", help="Securely transfer files to Android.")
    parser_transfer.add_argument("source", help="Path to the source file on Mac")
    parser_transfer.add_argument("--dest", help="Destination path on Android (default: /sdcard/Download/)", default="/sdcard/Download/")
    parser_transfer.add_argument("--force", action="store_true", help="Overwrite existing file on device")

    # Subcommand: drive-transfer (External Drive)
    parser_drive = subparsers.add_parser("drive-transfer", help="Transfer files to external drive (e.g., Toshiba).")
    parser_drive.add_argument("source", help="Path to the source file or folder")
    parser_drive.add_argument("--drive", help="Keyword to find drive (default: Toshiba)", default="Toshiba")
    parser_drive.add_argument("--dest-folder", help="Folder on drive to copy to", default="Transfer")
    parser_drive.add_argument("--dry-run", action="store_true", help="Simulate transfer")

    args = parser.parse_args()

    if args.command == "cleanup":
        run_cleanup(dry_run=args.dry_run)
    elif args.command == "transfer":
        success = run_transfer(args.source, args.dest, args.force)
        if not success:
            sys.exit(1)
    elif args.command == "drive-transfer":
        success = run_drive_transfer(args.source, args.drive, args.dest_folder, args.dry_run)
        if not success:
            sys.exit(1)
    else:
        parser.print_help()

if __name__ == "__main__":
    main()
