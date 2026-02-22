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

func allocGrid(h, w int) [][]float64 {
	g := make([][]float64, h)
	for i := range g {
		g[i] = make([]float64, w)
	}
	return g
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

	buf := allocGrid(fireH, fireW)
	sparks := allocGrid(fireH, fireW)

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
					buf = allocGrid(fireH, fireW)
					sparks = allocGrid(fireH, fireW)
					fmt.Print("\033[2J")
				}
			}

			// Hearth dimensions (also used for rendering base)
			hearthW := fireW * 2 / 3
			if hearthW > 80 {
				hearthW = 80
			}
			hearthLeft := (fireW - hearthW) / 2
			hearthRight := hearthLeft + hearthW
			hearthCenter := float64(hearthLeft+hearthRight) / 2.0
			hearthHalf := float64(hearthW) / 2.0

			// Seed heat along the hearth only
			for x := 0; x < fireW; x++ {
				if x < hearthLeft || x >= hearthRight {
					buf[fireH-1][x] = 0
					continue
				}
				dist := math.Abs(float64(x)-hearthCenter) / hearthHalf
				base := math.Exp(-dist * dist * 4.0)
				flicker := 0.15*math.Sin(float64(frame)*0.15+float64(x)*0.7) +
					0.1*math.Sin(float64(frame)*0.23+float64(x)*1.3)
				jitter := (rand.Float64() - 0.5) * 0.4
				if rand.Intn(5) == 0 {
					jitter -= 0.3
				}
				buf[fireH-1][x] = clamp(base+jitter+flicker, 0, 1)
			}

			// Occasional spark
			if rand.Intn(4) == 0 {
				sx := hearthLeft + rand.Intn(hearthW)
				sy := fireH - 3 - rand.Intn(fireH/3)
				if sy >= 0 && sy < fireH {
					sparks[sy][sx] = 0.6 + rand.Float64()*0.4
				}
			}

			// Propagate upward with narrow kernel
			for y := 0; y < fireH-1; y++ {
				for x := 0; x < fireW; x++ {
					sum := buf[y+1][x] * 2.0
					count := 2.0
					if x > 0 {
						sum += buf[y+1][x-1]
						count++
					}
					if x < fireW-1 {
						sum += buf[y+1][x+1]
						count++
					}
					avg := sum / count

					windDrift := math.Sin(float64(frame)*0.07 + float64(y)*0.4)
					drift := int(windDrift) + rand.Intn(3) - 1
					dx := clampInt(x+drift, 0, fireW-1)
					avg = (avg*2 + buf[y+1][dx]) / 3.0

					heightRatio := 1.0 - float64(y)/float64(fireH)
					cool := 0.12 + heightRatio*0.06 + rand.Float64()*0.06
					buf[y][x] = clamp(avg-cool, 0, 1)

					if sparks[y][x] > 0 {
						buf[y][x] = clamp(buf[y][x]+sparks[y][x], 0, 1)
						sparks[y][x] *= 0.5
						if sparks[y][x] < 0.05 {
							sparks[y][x] = 0
						}
						if y > 0 {
							sparks[y-1][x] = sparks[y][x] * 0.7
						}
					}
				}
			}

			// Render
			var sb strings.Builder
			sb.Grow(fireW * fireH * 2)
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
