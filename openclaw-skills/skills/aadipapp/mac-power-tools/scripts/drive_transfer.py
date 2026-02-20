import os
import shutil
import subprocess
import sys
from .utils import setup_logging

logger = setup_logging('drive_transfer')

def list_volumes():
    """Lists all mounted volumes in /Volumes."""
    if not os.path.exists("/Volumes"):
        return []
    return [d for d in os.listdir("/Volumes") if not d.startswith('.')]

def find_target_drive(name_keyword):
    """Finds a drive matching the keyword."""
    volumes = list_volumes()
    matches = [v for v in volumes if name_keyword.lower() in v.lower()]
    
    if not matches:
        return None
    
    if len(matches) > 1:
        logger.warning(f"Multiple drives found matching '{name_keyword}': {matches}. Using the first one.")
    
    return os.path.join("/Volumes", matches[0])

def check_writable(path):
    """Checks if the path is writable."""
    return os.access(path, os.W_OK)

def copy_data(source, dest, dry_run=False):
    """Copies file or directory to destination."""
    if dry_run:
        logger.info(f"[DRY RUN] Would copy '{source}' to '{dest}'")
        return True

    try:
        if os.path.isdir(source):
            if os.path.exists(dest):
                logger.warning(f"Destination '{dest}' already exists. Merging/Overwriting...")
                # shutil.copytree requires dest to NOT exist for python < 3.8 with dirs_exist_ok.
                # simpler to use cp -R for Mac specific skill or handle merge manually.
                # Let's use subprocess cp for robustness on Mac.
                subprocess.run(["cp", "-R", source, dest], check=True)
            else:
                shutil.copytree(source, dest)
        else:
            shutil.copy2(source, dest)
        
        logger.info(f"Successfully copied to '{dest}'")
        return True
    except Exception as e:
        logger.error(f"Copy failed: {e}")
        return False

def run_drive_transfer(source, drive_keyword="Toshiba", dest_folder="Transfer", dry_run=False):
    # 1. Validate Source
    source_path = os.path.abspath(source)
    if not os.path.exists(source_path):
        logger.error(f"Source not found: {source_path}")
        return False

    # 2. Find Drive
    logger.info(f"Searching for drive with keyword: '{drive_keyword}'...")
    drive_path = find_target_drive(drive_keyword)
    
    if not drive_path:
        logger.error(f"No drive found matching '{drive_keyword}'. Mounted volumes: {list_volumes()}")
        return False
    
    logger.info(f"Target drive found: {drive_path}")

    # 3. Check Writable
    if not check_writable(drive_path):
        logger.error(f"Drive '{drive_path}' is Read-Only! This happens often with NTFS drives on macOS.")
        logger.error("Solution: Install an NTFS driver (e.g., Paragon, Tuxera) or format drive to ExFAT.")
        return False

    # 4. Prepare Destination
    dest_path = os.path.join(drive_path, dest_folder, os.path.basename(source_path))
    parent_dest = os.path.dirname(dest_path)
    
    if not dry_run and not os.path.exists(parent_dest):
        try:
            os.makedirs(parent_dest, exist_ok=True)
        except OSError as e:
            logger.error(f"Failed to create destination folder '{parent_dest}': {e}")
            return False

    # 5. Perform Transfer
    logger.info(f"Transferring '{source_path}' -> '{dest_path}'")
    return copy_data(source_path, dest_path, dry_run=dry_run)
