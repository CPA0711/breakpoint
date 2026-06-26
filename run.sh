#!/bin/bash
set -e # Kalo error langsung stop, biar gak ngehang

echo "┏━╸┏━┓┏━┓   ┏┓ ┏━┓┏━╸┏━┓╻┏ ┏━┓┏━┓╻┏┓╻╺┳╸"
echo "┃  ┣━┛┣━┫   ┣┻┓┣┳┛┣╸ ┣━┫┣┻┓┣━┛┃ ┃┃┃┗┫ ┃ "
echo "┗━╸╹  ╹ ╹   ┗━┛╹┗╸┗━╸╹ ╹╹ ╹╹  ┗━┛╹╹ ╹ ╹ "
echo "  BUKAN BREAK MY HEART..."
echo ""

# Ambil url dari argumen
URL=""
for arg in "$@"; do
  case $arg in
    -url=*) URL="${arg#-url=}" ;;
  esac
done
# forward semua argumen ke go run
go run breakpoint.go "$@"
