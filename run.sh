#!/bin/bash
# Run di Termux

set -e 
CSV=breakpoint.csv
PNG=breakpoint.png
TXT=data.txt

echo "🚀 1. RUN TEST KE GOOGLE"
go run breakpoint.go -n 100 -c 10 -step=10s -warmer=3s -interval=100ms -url=https://google.com -csv=$CSV

echo "📝 2. ISI RPS MANUAL DARI OUTPUT DI ATAS"
cat > $TXT << EOF
1 4.83
2 9.30
3 10.03
4 9.18
5 10.15
6 10.01
7 10.15
8 10.02
9 10.16
10 10.00
EOF

echo "📈 3. PLOT GRAFIK"
gnuplot -e "set terminal pngcairo size 1200,600; set output '$PNG'; set title 'Breakpoint Test: RPS vs Concurrency'; set xlabel 'Concurrency C'; set ylabel 'RPS'; set grid; set key off; plot '$TXT' with linespoints lw 2 pt 7 ps 1.5"

echo "✅ SELESAI. Buka $PNG"
