#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Base URL
BASE_URL="http://localhost:8080"

echo -e "${GREEN}Starting API Test Script${NC}"

# Step 1: Login to get authentication tokens
echo -e "\n${GREEN}Step 1: Logging in...${NC}"
LOGIN_RESPONSE=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"email":"charlsoliv77@gmal.com","password":"chaRlsCha008"}' \
  "$BASE_URL/login" \
  -c cookies.txt)

echo "Login Response:"
echo $LOGIN_RESPONSE | jq '.'

# Step 2: Get all problems
echo -e "\n${GREEN}Step 2: Getting all problems...${NC}"
PROBLEMS_RESPONSE=$(curl -s -X GET \
  -b cookies.txt \
  "$BASE_URL/problems")

echo "Problems Response:"
echo $PROBLEMS_RESPONSE | jq '.'

# Extract first problem ID from the response
PROBLEM_ID=$(echo $PROBLEMS_RESPONSE | jq -r '.data[0].id')

if [ "$PROBLEM_ID" != "null" ] && [ ! -z "$PROBLEM_ID" ]; then
    echo -e "\n${GREEN}Step 3: Getting problem with ID: $PROBLEM_ID${NC}"
    
    # Step 3: Get specific problem by ID
    PROBLEM_RESPONSE=$(curl -s -X GET \
        -b cookies.txt \
        "$BASE_URL/problem/$PROBLEM_ID")

    echo "Problem Response:"
    echo $PROBLEM_RESPONSE | jq '.'
else
    echo -e "\n${RED}No problems found in the response${NC}"
fi

# Clean up
rm cookies.txt

echo -e "\n${GREEN}Test completed${NC}"