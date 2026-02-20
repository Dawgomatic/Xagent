import logging
import sys
import shutil
import subprocess

def setup_logging(name, level=logging.INFO):
    """
    Sets up a logger with the given name and level.
    """
    logger = logging.getLogger(name)
    logger.setLevel(level)
    
    # Check if handlers already exist to avoid duplicate logs
    if not logger.handlers:
        handler = logging.StreamHandler(sys.stdout)
        formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s')
        handler.setFormatter(formatter)
        logger.addHandler(handler)
        
    return logger

def check_command_installed(command):
    """
    Checks if a command-line tool is installed and available in the PATH.
    """
    return shutil.which(command) is not None

def run_subprocess(command, check=True, capture_output=True, text=True):
    """
    Wrapper around subprocess.run with common defaults.
    """
    try:
        result = subprocess.run(
            command, 
            check=check, 
            capture_output=capture_output, 
            text=text
        )
        return result
    except subprocess.CalledProcessError as e:
        # Re-raise or handle, here we re-raise to let caller decide
        raise e
