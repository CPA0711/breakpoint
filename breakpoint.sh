#!/data/data/com.termux/files/usr/bin/sh
cd ~
clear
echo "🔥 BREAKPOINT v1.0 🔥"
echo "CPA Ramp-Up Load Tester"
echo "--------------------------------------"
echo "Target: https://google.com"
echo "CSV: /sdcard/breakpoint.csv"
echo ""

go run breakpoint.go -n 300 -c 10 -step=20s -warmer=5s -interval=100ms -url=https://google.com -csv=/sdcard/breakpoint.csv

echo ""
echo "✅ SELESAI"
echo "📁 Cek file: /sdcard/breakpoint.csv"
termux-vibrate -d 500
termux-notification --title "BreakPoint Selesai" --content "Cek /sdcard/breakpoint.csv"
