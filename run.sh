#!/bin/bash
# CPA BREAKPOINT - Cek RPS Limit Google

set -e 
CSV=breakpoint.csv

# BANNER ASCII - Versi Aman No Kutip
echo "  _______  _____  ______    _______  ______   _______  _______  "
echo " / ____/ |/ / _ ) / __/   / ____/ / __/ /  /  _/ _ \ / ____/  "
echo "/ /      |   / _  \ \/    /     / _// /___/ // , _// / __    "
echo "\____/   |__//____/___/   /_/___/ /___/____/___/_/|_|/___/    "
echo "        CPA BREAKPOINT - By CPA0711"
echo ""

echo "🚀 JALANIN TEST KE GOOGLE..."
go run breakpoint.go -n 100 -c 10 -step=10s -warmer=3s -interval=100ms -url=https://google.com -csv=$CSV

echo ""
echo "✅ SELESAI. CSV ada di: $CSV"
echo "📌 Liat bagian 'SUMMARY' di atas. Titik mentoknya = Breakpoint"
