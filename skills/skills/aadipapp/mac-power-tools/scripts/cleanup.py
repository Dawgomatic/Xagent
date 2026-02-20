import os
import shutil
import time
from .utils import setup_logging

logger = setup_logging('mac_cleanup')

def get_file_age_days(filepath):
    """Returns the age of the file in days."""
    return (time.time() - os.path.getmtime(filepath)) / (24 * 3600)

def cleanup_directory(directory, max_age_days=0, dry_run=True):
    """
    Cleans up files in a directory.
    If max_age_days > 0, only deletes files older than that.
    """
    if not os.path.exists(directory):
        logger.warning(f"Directory not found: {directory}")
        return 0, 0

    logger.info(f"Scanning {directory}...")
    count = 0
    size_freed = 0

    try:
        for item in os.listdir(directory):
            item_path = os.path.join(directory, item)
            
            try:
                if os.path.isfile(item_path):
                    age = get_file_age_days(item_path)
                    if max_age_days > 0 and age < max_age_days:
                        continue
                    
                    size_freed += os.path.getsize(item_path)
                    if dry_run:
                        logger.info(f"[DRY RUN] Would delete file: {item_path} (Age: {age:.1f} days)")
                    else:
                        os.remove(item_path)
                        logger.info(f"Deleted file: {item_path}")
                    count += 1
                
                elif os.path.isdir(item_path):
                    if directory.endswith(".Trash"):
                         # In trash, delete directories too
                        if dry_run:
                            logger.info(f"[DRY RUN] Would delete directory: {item_path}")
                        else:
                            shutil.rmtree(item_path)
                            logger.info(f"Deleted directory: {item_path}")
                        count += 1
                    
            except Exception as e:
                logger.error(f"Error processing {item_path}: {e}")

    except PermissionError:
        logger.error(f"Permission denied accessing {directory}")

    logger.info(f"Found {count} items to clean in {directory}. Est. space: {size_freed / (1024*1024):.2f} MB")
    return count, size_freed

def run_cleanup(dry_run=True):
    if not dry_run:
        print("WARNING: You are about to permanently delete files.")
        choice = input("Are you sure you want to proceed? (y/N): ").lower()
        if choice != 'y':
            logger.info("Cleanup cancelled by user.")
            return

    logger.info(f"Starting Cleanup (Dry Run: {dry_run})")
    
    total_files = 0
    total_size = 0

    # 1. Clean Trash
    trash_dir = os.path.expanduser("~/.Trash")
    c, s = cleanup_directory(trash_dir, dry_run=dry_run)
    total_files += c
    total_size += s

    # 2. Clean Caches
    caches_dir = os.path.expanduser("~/Library/Caches")
    logger.warning("Excessive cache cleaning can slow down system temporarily.")
    c, s = cleanup_directory(caches_dir, max_age_days=7, dry_run=dry_run)
    total_files += c
    total_size += s

    # 3. Clean Downloads
    downloads_dir = os.path.expanduser("~/Downloads")
    c, s = cleanup_directory(downloads_dir, max_age_days=30, dry_run=dry_run)
    total_files += c
    total_size += s

    logger.info(f"Cleanup Complete. Total items: {total_files}, Total freed: {total_size / (1024*1024):.2f} MB")
