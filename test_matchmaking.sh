#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

# Function to print status
print_status() {
    echo -e "${BLUE}[*] $1${NC}"
}

print_success() {
    echo -e "${GREEN}[+] $1${NC}"
}

# Clean up any existing cookie files
print_status "Cleaning up old cookie files..."
rm -f cookies.txt cookies2.txt

# Login first user
print_status "Logging in first user..."
curl -X POST http://localhost:8080/login \
    -H "Content-Type: application/json" \
    -d '{"email": "jhondoe@gmail.com", "password": "iAmJhonDoe007"}' \
    -c cookies.txt -s > /dev/null
print_success "First user logged in"

# Login second user
print_status "Logging in second user..."
curl -X POST http://localhost:8080/login \
    -H "Content-Type: application/json" \
    -d '{"email": "charlsoliv77@gmal.com", "password": "chaRlsCha008"}' \
    -c cookies2.txt -s > /dev/null
print_success "Second user logged in"

# Extract access tokens
print_status "Extracting access tokens..."
ACCESS_TOKEN=$(grep access_token cookies.txt | awk '{print $7}')
ACCESS_TOKEN_2=$(grep access_token cookies2.txt | awk '{print $7}')

# Function to start WebSocket connection
start_websocket() {
    local token=$1
    local user=$2
    print_status "Starting WebSocket connection for $user..."
    wscat -c "ws://localhost:8080/ws" \
        -H "Cookie: access_token=$token"
}

# Create a menu
echo -e "\n${BLUE}Code War Game Test Script${NC}"
echo "1. Start first player (jhondoe@gmail.com)"
echo "2. Start second player (charlsoliv77@gmal.com)"
echo "3. Start both players (in separate terminals)"
echo "4. Clean up and exit"
echo -n "Choose an option (1-4): "

read choice

case $choice in
    1)
        start_websocket "$ACCESS_TOKEN" "first player"
        ;;
    2)
        start_websocket "$ACCESS_TOKEN_2" "second player"
        ;;
    3)
        print_status "Starting both players..."
        # Start first player in new terminal using xterm
        xterm -e "echo 'Starting first player...'; wscat -c 'ws://localhost:8080/ws' -H 'Cookie: access_token=$ACCESS_TOKEN'" &
        # Start second player in new terminal using xterm
        xterm -e "echo 'Starting second player...'; wscat -c 'ws://localhost:8080/ws' -H 'Cookie: access_token=$ACCESS_TOKEN_2'" &
        ;;
    4)
        print_status "Cleaning up..."
        rm -f cookies.txt cookies2.txt
        print_success "Cleanup complete"
        exit 0
        ;;
    *)
        echo "Invalid option"
        exit 1
        ;;
esac