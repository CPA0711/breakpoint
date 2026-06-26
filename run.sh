#!/bin/bash
set -e

echo "CPA BREAKPOINT - By CPA0711"
echo ""
echo "🚀 STARTING BREAKPOINT..."
echo "Args: $@"
echo ""

go run breakpoint.go "$@"
