#!/usr/bin/env python3
"""
Strava OAuth authentication flow
Gets access token and refresh token for API calls
"""

import os
import json
import urllib.request
import urllib.parse
from http.server import HTTPServer, BaseHTTPRequestHandler
import threading
import webbrowser

# Config
CLIENT_ID = os.environ.get('STRAVA_CLIENT_ID')
CLIENT_SECRET = os.environ.get('STRAVA_CLIENT_SECRET')
REDIRECT_URI = 'http://localhost:8080/callback'
TOKEN_FILE = os.path.expanduser('~/.strava_tokens.json')

if not CLIENT_ID or not CLIENT_SECRET:
    print("Error: Set STRAVA_CLIENT_ID and STRAVA_CLIENT_SECRET environment variables")
    exit(1)

class CallbackHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if '/callback' in self.path:
            # Parse authorization code
            query = urllib.parse.urlparse(self.path).query
            params = urllib.parse.parse_qs(query)
            
            if 'code' in params:
                self.server.auth_code = params['code'][0]
                self.send_response(200)
                self.end_headers()
                self.wfile.write(b'Authorization successful! You can close this window.')
            elif 'error' in params:
                self.server.error = params['error'][0]
                self.send_response(400)
                self.end_headers()
                self.wfile.write(f'Error: {params["error"][0]}'.encode())
        else:
            self.send_response(404)
            self.end_headers()
    
    def log_message(self, format, *args):
        pass  # Suppress HTTP logs

def get_authorization_url():
    """Build OAuth authorization URL"""
    scopes = 'read,activity:read'  # Read your activities
    params = {
        'client_id': CLIENT_ID,
        'redirect_uri': REDIRECT_URI,
        'response_type': 'code',
        'approval_prompt': 'force',
        'scope': scopes
    }
    return f"https://www.strava.com/oauth/authorize?{urllib.parse.urlencode(params)}"

def exchange_code_for_token(code):
    """Exchange authorization code for access token"""
    url = 'https://www.strava.com/oauth/token'
    data = {
        'client_id': CLIENT_ID,
        'client_secret': CLIENT_SECRET,
        'code': code,
        'grant_type': 'authorization_code'
    }
    
    req = urllib.request.Request(
        url,
        data=urllib.parse.urlencode(data).encode(),
        headers={'Content-Type': 'application/x-www-form-urlencoded'},
        method='POST'
    )
    
    with urllib.request.urlopen(req) as response:
        return json.loads(response.read().decode())

def save_tokens(token_data):
    """Save tokens to file"""
    with open(TOKEN_FILE, 'w') as f:
        json.dump(token_data, f, indent=2)
    print(f"Tokens saved to {TOKEN_FILE}")

def main():
    # Start local server to receive callback
    server = HTTPServer(('localhost', 8080), CallbackHandler)
    server.auth_code = None
    server.error = None
    
    server_thread = threading.Thread(target=server.serve_forever)
    server_thread.daemon = True
    server_thread.start()
    
    # Open browser for authorization
    auth_url = get_authorization_url()
    print(f"Opening browser for authorization...")
    print(f"If browser doesn't open, visit: {auth_url}")
    webbrowser.open(auth_url)
    
    # Wait for callback
    print("Waiting for authorization...")
    while server.auth_code is None and server.error is None:
        pass
    
    server.shutdown()
    
    if server.error:
        print(f"Authorization failed: {server.error}")
        return
    
    # Exchange code for token
    print("Getting access token...")
    token_data = exchange_code_for_token(server.auth_code)
    
    # Save tokens
    save_tokens(token_data)
    
    print(f"\nAuthenticated as: {token_data.get('athlete', {}).get('firstname')} {token_data.get('athlete', {}).get('lastname')}")
    print(f"Access token: {token_data.get('access_token')[:20]}...")
    print("\nYou can now subscribe to webhooks!")

if __name__ == '__main__':
    main()
