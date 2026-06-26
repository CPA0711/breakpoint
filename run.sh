#!/bin/bash
set -e # Kalo error langsung stop, biar gak ngehang

echo "┏━┛┏━┃┏━┃  ┏━ ┏━┃┏━┛┏━┃┃ ┃┏━┃┏━┃┛┏━ ━┏┛"
echo "┃  ┏━┛┏━┃ ┏━┃┏┏┛┏━┛┏━┃┏┛ ┏━┛┃ ┃┃┃ ┃ ┃    "
echo " ━┛┛  ┛ ┛  ━━ ┛ ┛━━┛┛┛┛ ┛ ┛ ━   ━┛┛ ┛ ┛ ┛    "
echo "        CPA BREAKPOINT "
echo ""

# Ambil url dari argumen
URL=""
for arg in "$@"; do
  case $arg in
    -url=*) URL="${arg#-url=}" ;;
  esac
done

echo "🚀 STARTING BREAKPOINT..."
echo "Target: ${URL:-TIDAK DISET} | $@"
echo ""

# forward semua argumen ke go run
go run breakpoint.go "$@"
