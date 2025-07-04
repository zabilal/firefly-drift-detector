#!/bin/bash

# Test script for drift detection with mock data
# This script runs the drift detector against different test cases

set -e

# Function to run a test case
run_test() {
    local test_name=$1
    local instance_id=$2
    local mock_file=$3
    
    echo "\n=== Running test: $test_name ==="
    echo "Instance ID: $instance_id"
    echo "Mock file: $mock_file"
    echo "----------------------------------------"
    
    go run cmd/main.go detect \
        --instance "$instance_id" \
        --tf-dir testdata/terraform \
        --mock \
        --mock-file "$mock_file"
}

# Test 1: Instance Type Drift
echo "\n\n===== TEST 1: Instance Type Drift ====="
run_test "Instance Type Drift" "i-1234567890abcdef1" "testdata/mock_tests/instance_type_drift.json"

# Test 2: Security Group Drift
echo "\n\n===== TEST 2: Security Group Drift ====="
run_test "Security Group Drift" "i-1234567890abcdef2" "testdata/mock_tests/security_group_drift.json"

# Test 3: Tag Drift
echo "\n\n===== TEST 3: Tag Drift ====="
run_test "Tag Drift" "i-1234567890abcdef3" "testdata/mock_tests/tag_drift.json"

echo "\n\n===== TESTS COMPLETE ====="
