#!/bin/bash
echo "       "
echo "╔═╝╔═║╔═║  ╔═ ╔═║╔═╝╔═║ ║╔═║╔═║╝╔═ ═╔╝"
echo "║  ╔═╝╔═║  ╔═║╔╝╔═╝╔═║╔╝ ╔═╝║ ║║ ║"
echo "══╝  ╝  ══ ╝══╝ ╝╝ ╝╝  ══╝ ╝"
echo "        CPA BREAKPOINT - By CPA0711"
echo ""

# Gabungin semua argumen jadi 1 string biar formatnya -key=value
ARGS=""
for arg in "$@"; do
  ARGS="$ARGS $arg"
done

echo "🚀 STARTING BREAKPOINT..."
echo "Target: $1 $2 | Max C: $3 $4 | N: $5 $6"
echo ""

# KUNCINYA DI SINI: go run breakpoint.go $ARGS
go run breakpoint.go $ARGS
