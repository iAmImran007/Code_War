#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Create or clear the cookies file
echo "User Authentication Cookies" > auth_cookies.txt
echo "=========================" >> auth_cookies.txt
echo "" >> auth_cookies.txt

echo "Creating test users..."

# First user - John Doe
echo -e "\n${GREEN}Creating user: jhondoe@gmail.com${NC}"
COOKIE1=$(curl -X POST http://localhost:8080/signup \
  -H "Content-Type: application/json" \
  -d '{"email": "jhondoe@gmail.com", "password": "iAmJhonDoe007"}' \
  -i | grep -i "Set-Cookie" | cut -d ';' -f1 | cut -d ' ' -f2)

# Save first user's cookie
echo "John Doe (jhondoe@gmail.com):" >> auth_cookies.txt
echo "Cookie: $COOKIE1" >> auth_cookies.txt
echo "" >> auth_cookies.txt

# Second user - Charles Oliver
echo -e "\n${GREEN}Creating user: charlsoliv77@gmal.com${NC}"
COOKIE2=$(curl -X POST http://localhost:8080/signup \
  -H "Content-Type: application/json" \
  -d '{"email": "charlsoliv77@gmal.com", "password": "chaRlsCha008"}' \
  -i | grep -i "Set-Cookie" | cut -d ';' -f1 | cut -d ' ' -f2)

# Save second user's cookie
echo "Charles Oliver (charlsoliv77@gmal.com):" >> auth_cookies.txt
echo "Cookie: $COOKIE2" >> auth_cookies.txt

echo -e "\n${GREEN}Test users creation completed!${NC}"
echo -e "${GREEN}Cookies have been saved to auth_cookies.txt${NC}"