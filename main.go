package main

import (
"fmt"
"math"
"math/rand"
"os"
"os/signal"
"strings"
"syscall"
"time"
"unsafe"
)

// ASCII fire characters from hottest to coolest
var fireChars = []byte{'#', '@', '%', '&', '*', '+', '=', ':', '~', '-', '.', ' '}

func termSize() (int, int) {
type winsize struct {
Row, Col, Xpixel, Ypixel uint16
}
var ws winsize
_, _, _ = syscall.Syscall(
syscall.SYS_IOCTL,
uintptr(syscall.Stdout),
uintptr(syscall.TIOCGWINSZ),
uintptr(unsafe.Pointer(&ws)),
)
w, h := int(ws.Col), int(ws.Row)
if w == 0 {
w = 80
}
if h == 0 {
h = 24
}
return w, h
}

func main() {
sig := make(chan os.Signal, 1)
signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

termW, termH := termSize()

fireW := termW
fireH := termH - 4
if fireH < 10 {
fireH = 10
}

fmt.Print("\033[?25l\033[2J")
defer fmt.Print("\033[?25h\033[0m\033[2J")

buf := make([][]float64, fireH)
for i := range buf {
buf[i] = make([]float64, fireW)
}

sparks := make([][]float64, fireH)
for i := range sparks {
sparks[i] = make([]float64, fireW)
}

ticker := time.NewTicker(50 * time.Millisecond)
defer ticker.Stop()

frame := 0
for {
select {
case <-sig:
return
case <-ticker.C:
frame++

if frame%20 == 0 {
newW, newH := termSize()
if newW != termW || newH != termH {
termW, termH = newW, newH
fireW = termW
fireH = termH - 4
if fireH < 10 {
fireH = 10
}
buf = make([][]float64, fireH)
for i := range buf {
buf[i] = make([]float64, fireW)
}
sparks = make([][]float64, fireH)
for i := range sparks {
sparks[i] = make([]float64, fireW)
}
fmt.Print("\033[2J")
}
}

center := float64(fireW) / 2.0
for x := 0; x < fireW; x++ {
dist := math.Abs(float64(x)-center) / center
base := math.Exp(-dist * dist * 3.0)
pulse := 0.05 * math.Sin(float64(frame)*0.1+float64(x)*0.3)
jitter := (rand.Float64() - 0.5) * 0.3
buf[fireH-1][x] = clamp(base+jitter+pulse, 0, 1)

if fireH >= 3 {
buf[fireH-2][x] = clamp(base*0.9+jitter*0.8+pulse, 0, 1)
}
}

if rand.Intn(3) == 0 {
sx := int(center) + rand.Intn(fireW/3) - fireW/6
sy := fireH - 3 - rand.Intn(fireH/4)
if sx >= 0 && sx < fireW && sy >= 0 && sy < fireH {
sparks[sy][sx] = 0.8 + rand.Float64()*0.2
}
}

for y := 0; y < fireH-2; y++ {
for x := 0; x < fireW; x++ {
sum := 0.0
count := 0.0
for dx := -2; dx <= 2; dx++ {
nx := x + dx
if nx >= 0 && nx < fireW {
weight := 1.0
if dx == 0 {
weight = 3.0
} else if dx == -1 || dx == 1 {
weight = 2.0
}
sum += buf[y+1][nx] * weight
count += weight
if y+2 < fireH {
sum += buf[y+2][nx] * weight * 0.5
count += weight * 0.5
}
}
}
avg := sum / count

windDrift := math.Sin(float64(frame)*0.07+float64(y)*0.5) * 1.5
drift := int(windDrift) + rand.Intn(3) - 1
dx := clampInt(x+drift, 0, fireW-1)
avg = (avg*3 + buf[y+1][dx]) / 4.0

heightFactor := float64(fireH-y) / float64(fireH)
cool := 0.06 + heightFactor*0.08 + rand.Float64()*0.04
buf[y][x] = clamp(avg-cool, 0, 1)

if sparks[y][x] > 0 {
buf[y][x] = clamp(buf[y][x]+sparks[y][x], 0, 1)
sparks[y][x] *= 0.6
if sparks[y][x] < 0.05 {
sparks[y][x] = 0
}
if y > 0 {
sparks[y-1][x] = sparks[y][x] * 0.8
}
}
}
}

var sb strings.Builder
sb.Grow(fireW * fireH * 4)
sb.WriteString("\033[H")

for y := 0; y < fireH; y++ {
for x := 0; x < fireW; x++ {
heat := buf[y][x]
if heat < 0.02 {
sb.WriteByte(' ')
continue
}
fi := int((1.0 - heat) * float64(len(fireChars)-1))
fi = clampInt(fi, 0, len(fireChars)-1)
sb.WriteByte(fireChars[fi])
}
sb.WriteByte('\n')
}

// Hearth base
hearthW := fireW * 2 / 3
if hearthW > 80 {
hearthW = 80
}
pad := (fireW - hearthW) / 2

sb.WriteString(strings.Repeat(" ", pad))
sb.WriteString(strings.Repeat("_", hearthW))
sb.WriteByte('\n')

sb.WriteString(strings.Repeat(" ", pad))
emberChars := []byte{'.', ',', ':', ';', '\'', '`'}
for i := 0; i < hearthW; i++ {
sb.WriteByte(emberChars[rand.Intn(len(emberChars))])
}
sb.WriteByte('\n')

sb.WriteString(strings.Repeat(" ", pad))
sb.WriteString(strings.Repeat("=", hearthW))
sb.WriteByte('\n')

fmt.Print(sb.String())
}
}
}

func clamp(v, lo, hi float64) float64 {
if v < lo {
return lo
}
if v > hi {
return hi
}
return v
}

func clampInt(v, lo, hi int) int {
if v < lo {
return lo
}
if v > hi {
return hi
}
return v
}
