#!/bin/bash
set -e 
CSV=breakpoint.csv

URL="https://google.com"
N=100
C=10

# BACA ARGUMEN: support -url=... DAN -url ...
while [[ $# -gt 0 ]]; do
  case $1 in
    -url=*) URL="${1#*=}"; shift ;;
    -url) URL="$2"; shift 2 ;;
    -n=*) N="${1#*=}"; shift ;;
    -n) N="$2"; shift 2 ;;
    -c=*) C="${1#*=}"; shift ;;
    -c) C="$2"; shift 2 ;;
    *) shift ;;
  esac
done

# BANNER ASCII - Tetep ada
cat <<'EOF'
╔═╝╔═║╔═║  ╔═ ╔═║╔═╝╔═║ ║╔═║╔═║╝╔═ ═╔╝
║  ╔═╝╔═║  ╔═║╔╝╔═╝╔═║╔╝ ╔═╝║ ║║ ║ 
══╝  ╝ ╝  ══ ╝══╝ ╝╝ ╝╝  ══╝ ╝ 
        CPA BREAKPOINT - By CPA0711
EOF
echo ""
echo "🚀 STARTING BREAKPOINT..."
echo "Target: $URL | Max C: $C | N: $N"

# Trik biar gak kecampur PS1 Zsh: sembunyiin progress bar
go run breakpoint.go -n $N -c $C -step=10s -warmer=3s -interval=100ms -url=$URL -csv=$CSV | grep -E "SUMMARY|BREAKPOINT SELESAI"

echo ""
echo "✅ DONE... CSV at: $CSV"
