#!/bin/bash
# CPA BREAKPOINT

set -e 
CSV=breakpoint.csv

# DEFAULT VALUE
URL="https://google.com"
N=100
C=10

# BACA ARGUMEN: -url=... -n=... -c=...
for arg in "$@"; do
  case $arg in
    -url=*) URL="${arg#*=}" ;;
    -n=*) N="${arg#*=}" ;;
    -c=*) C="${arg#*=}" ;;
  esac
done

# BANNER ASCII - Aman pake Heredoc
cat <<'EOF'
  _______  _____  ______    _______  ______   _______  _______  
 / ____/ |/ _ ) / __/   / ____/ / __/ /  /  _/ _ \ / ____/  
/ /      |   / _  \ \/    /     / _// /___/ // , _// / __    
\____/   |__//____/___/   /_/___/ /___/____/___/_/|_|/___/    
        CPA BREAKPOINT - By CPA0711
EOF
echo ""

echo "🚀 STARTING BREAKPOINT..."
echo "Target: $URL | Max C: $C | N: $N"
go run breakpoint.go -n $N -c $C -step=10s -warmer=3s -interval=100ms -url=$URL -csv=$CSV

echo ""
echo "✅ SELESAI. CSV ada di: $CSV"
