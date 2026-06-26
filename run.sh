#!/bin/bash
set -e # Kalo error langsung stop, biar gak ngehang

echo "╔═╝╔═║╔═║  ╔═ ╔═║╔═╝╔═║ ║╔═║╔═║╝╔═ ═╔╝"
echo "║  ╔═╝╔═║  ╔═║╔╝╔═╝╔═║╔╝ ╔═╝║ ║║ ║"
echo "══╝  ╝  ══ ╝══╝ ╝╝ ╝╝  ══╝ ╝"
echo "        CPA BREAKPOINT - By CPA0711"
echo ""

# Ambil url dari argumen buat di print doang
URL=""
for arg in "$@"; do
  case $arg in
    -url=*) URL="${arg#-url=}" ;;
  esac
done

echo "🚀 STARTING BREAKPOINT..."
echo "Target: ${URL:-TIDAK DISET} | Args: $@"
echo ""

# KUNCINYA: Langsung forward semua argumen ke go run
go run breakpoint.go "$@"
