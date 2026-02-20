import os
import sys
import hashlib
import shlex
import subprocess
from .utils import setup_logging, check_command_installed

logger = setup_logging('android_transfer')

def calculate_sha256(filepath):
    """Calculates the SHA256 checksum of a file."""
    sha256_hash = hashlib.sha256()
    with open(filepath, "rb") as f:
        # Read and update hash string value in blocks of 4K
        for byte_block in iter(lambda: f.read(4096), b""):
            sha256_hash.update(byte_block)
    return sha256_hash.hexdigest()

def check_device_authorized():
    """Checks if a device is connected and authorized."""
    try:
        if not check_command_installed("adb"):
            logger.error("'adb' is not found. Please install Android Platform Tools.")
            return False

        result = subprocess.run(["adb", "devices"], capture_output=True, text=True, check=True)
        lines = result.stdout.strip().split('\n')[1:] # Skip header
        valid_devices = [line for line in lines if "device" in line and "offline" not in line and "unauthorized" not in line]
        
        if not valid_devices:
            logger.error("No authorized Android device found. Please connect your device and enable USB Debugging.")
            return False
        
        if len(valid_devices) > 1:
            logger.warning(f"Multiple devices found. Using the first one: {valid_devices[0].split()[0]}")
        
        return True
    except subprocess.CalledProcessError:
        return False

def sanitize_path(path):
    """Sanitizes user input path to prevent traversal."""
    if ".." in path:
        raise ValueError("Invalid path: Directory traversal ('..') is not allowed.")
    return path

import re

# ... existing imports ...

def validate_filename(filename):
    """Ensures filename only contains safe characters."""
    if not re.match(r'^[a-zA-Z0-9_.-]+$', filename):
        raise ValueError(f"Invalid filename '{filename}'. Only alphanumeric, dot, underscore, and dash allowed.")

def validate_dest_path(path):
    """Ensures destination is in a safe allowed directory."""
    allowed_prefixes = ["/sdcard/", "/storage/", "/data/local/tmp/"]
    if not any(path.startswith(prefix) for prefix in allowed_prefixes):
        raise ValueError(f"Destination path '{path}' is not in an allowed directory ({', '.join(allowed_prefixes)}).")

def run_transfer(source, dest, force=False):
    # 1. Environment Check
    if not check_device_authorized():
        return False

    # 2. Path Validation & Sanitization
    try:
        source_path = os.path.abspath(sanitize_path(source))
        
        # Security: Symlink Check
        if os.path.islink(source_path):
            logger.error(f"Security Error: Source '{source_path}' is a symlink. Transferring symlinks is prohibited.")
            return False

        if not os.path.exists(source_path):
             logger.error(f"Source file does not exist: {source_path}")
             return False
        
        # Security: Filename Validation
        filename = os.path.basename(source_path)
        validate_filename(filename)

        dest_dir = dest
        if not dest_dir.endswith("/"):
            dest_dir += "/"
        
        if not dest_dir.startswith("/"):
             logger.error("Destination path must be absolute (start with /).")
             return False
        
        # Security: Destination Whitelist
        validate_dest_path(dest_dir)

        dest_path = dest_dir + filename
        
    except ValueError as e:
        logger.error(f"Security Error: {e}")
        return False

    # 3. Overwrite Protection
    logger.info(f"Checking destination: {dest_path}...")
    check_cmd = ["adb", "shell", "[ -f " + shlex.quote(dest_path) + " ] && echo 'EXISTS'"]
    check_res = subprocess.run(check_cmd, capture_output=True, text=True)
    if "EXISTS" in check_res.stdout and not force:
        logger.error(f"File already exists at {dest_path}. Use --force to overwrite.")
        return False

    # 4. Calculate Local Checksum
    logger.info(f"Calculating local SHA256 for {source_path}...")
    local_sha256 = calculate_sha256(source_path)
    logger.info(f"Local SHA256: {local_sha256}")

    # 5. Transfer
    logger.info(f"Transferring to {dest_path}...")
    push_res = subprocess.run(["adb", "push", source_path, dest_path], capture_output=True, text=True)
    if push_res.returncode != 0:
        logger.error(f"Transfer Failed: {push_res.stderr}")
        return False

    # 6. Verify Integrity
    logger.info("Verifying remote checksum...")
    remote_res = subprocess.run(["adb", "shell", "sha256sum", shlex.quote(dest_path)], capture_output=True, text=True)
    
    remote_hash = None
    if remote_res.returncode == 0 and remote_res.stdout.strip():
        remote_hash = remote_res.stdout.split()[0]
    else:
        # Check if sha256sum was missing
        if "not found" in remote_res.stderr:
            logger.error("Device does not support sha256sum verification. Cannot guarantee highest security.")
            # Cleanup
            subprocess.run(["adb", "shell", "rm", shlex.quote(dest_path)])
            return False
    
    if remote_hash == local_sha256:
        logger.info("Success! File transferred securely and verified.")
        return True
    else:
        logger.critical("Checksum mismatch!")
        logger.critical(f"Local:  {local_sha256}")
        logger.critical(f"Remote: {remote_hash}")
        logger.info("Deleting corrupted file from device...")
        subprocess.run(["adb", "shell", "rm", shlex.quote(dest_path)])
        return False
