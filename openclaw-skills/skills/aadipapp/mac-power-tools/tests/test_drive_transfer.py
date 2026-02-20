import unittest
from unittest.mock import MagicMock, patch
import sys
import os

# Add skill root to path
sys.path.append(os.path.join(os.path.dirname(__file__), '..'))

from scripts.drive_transfer import find_target_drive, check_writable

class TestDriveTransfer(unittest.TestCase):
    @patch('os.listdir')
    @patch('os.path.exists')
    def test_find_drive(self, mock_exists, mock_listdir):
        mock_exists.return_value = True
        mock_listdir.return_value = ['Macintosh HD', 'Toshiba Ext', 'Recovery']
        
        drive = find_target_drive("Toshiba")
        self.assertEqual(drive, "/Volumes/Toshiba Ext")
    
    @patch('os.listdir')
    def test_find_drive_not_found(self, mock_listdir):
        mock_listdir.return_value = ['Macintosh HD']
        drive = find_target_drive("Toshiba")
        self.assertIsNone(drive)

    @patch('os.access')
    def test_check_writable(self, mock_access):
        mock_access.return_value = True
        self.assertTrue(check_writable("/Volumes/Toshiba"))
        
        mock_access.return_value = False
        self.assertFalse(check_writable("/Volumes/Toshiba"))

if __name__ == '__main__':
    unittest.main()
