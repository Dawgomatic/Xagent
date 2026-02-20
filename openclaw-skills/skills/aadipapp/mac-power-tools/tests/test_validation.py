import unittest
import sys
import os

# Add skill root to path
sys.path.append(os.path.join(os.path.dirname(__file__), '..'))

from scripts.transfer import validate_filename, validate_dest_path

class TestValidation(unittest.TestCase):
    def test_filename_valid(self):
        validate_filename("valid_file.txt")
        validate_filename("my-file_1.png")
    
    def test_filename_invalid(self):
        with self.assertRaises(ValueError):
            validate_filename("file;rm -rf /")
        with self.assertRaises(ValueError):
            validate_filename("../file")

    def test_dest_valid(self):
        validate_dest_path("/sdcard/Download/")
        validate_dest_path("/storage/emulated/0/")
        validate_dest_path("/data/local/tmp/test")

    def test_dest_invalid(self):
        with self.assertRaises(ValueError):
            validate_dest_path("/system/bin/")
        with self.assertRaises(ValueError):
            validate_dest_path("/etc/")

if __name__ == '__main__':
    unittest.main()
