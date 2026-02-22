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

// Denser chars at bottom, wispy at top
var fireChars = []byte{'#', '#', '@', '%', '&', '*', '+', '=', ':', '~', '-', '.', '`', ' '}

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

	// Persistent flame tongue positions and intensities
	type tongue struct {
		x         float64
		intensity float64
		speed     float64
		phase     float64
		width     float64
	}

	hearthW := fireW * 2 / 3
	if hearthW > 80 {
		hearthW = 80
	}
	hearthLeft := (fireW - hearthW) / 2

	// Create several flame tongues across the hearth
	numTongues := hearthW / 4
	if numTongues < 5 {
		numTongues = 5
	}
	tongues := make([]tongue, numTongues)
	for i := range tongues {
		tongues[i] = tongue{
			x:         float64(hearthLeft) + float64(i)*float64(hearthW)/float64(numTongues) + rand.Float64()*3,
			intensity: 0.5 + rand.Float64()*0.5,
			speed:     0.08 + rand.Float64()*0.12,
			phase:     rand.Float64() * math.Pi * 2,
			width:     1.5 + rand.Float64()*2.5,
		}
	}

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
					hearthW = fireW * 2 / 3
					if hearthW > 80 {
						hearthW = 80
					}
					hearthLeft = (fireW - hearthW) / 2
					buf = allocGrid(fireH, fireW)
					sparks = allocGrid(fireH, fireW)
					numTongues = hearthW / 4
					if numTongues < 5 {
						numTongues = 5
					}
					tongues = make([]tongue, numTongues)
					for i := range tongues {
						tongues[i] = tongue{
							x:         float64(hearthLeft) + float64(i)*float64(hearthW)/float64(numTongues) + rand.Float64()*3,
							intensity: 0.5 + rand.Float64()*0.5,
							speed:     0.08 + rand.Float64()*0.12,
							phase:     rand.Float64() * math.Pi * 2,
							width:     1.5 + rand.Float64()*2.5,
						}
					}
					fmt.Print("\033[2J")
				}
			}

			hearthRight := hearthLeft + hearthW

			// Clear bottom rows
			for x := 0; x < fireW; x++ {
				buf[fireH-1][x] = 0
			}

			// Each tongue contributes a gaussian heat bump
			t := float64(frame)
			for _, tng := range tongues {
				// Tongues sway side to side
				sway := math.Sin(t*tng.speed+tng.phase) * 2.0
				cx := tng.x + sway
				// Pulsing intensity
				pulse := tng.intensity * (0.7 + 0.3*math.Sin(t*tng.speed*1.7+tng.phase))

				for x := hearthLeft; x < hearthRight; x++ {
					dist := (float64(x) - cx) / tng.width
					heat := pulse * math.Exp(-dist*dist)
					buf[fireH-1][x] = clamp(buf[fireH-1][x]+heat, 0, 1)
				}
			}

			// Add some baseline warmth across hearth and jitter
			for x := hearthLeft; x < hearthRight; x++ {
				dist := math.Abs(float64(x)-float64(hearthLeft+hearthW/2)) / float64(hearthW/2)
				base := 0.3 * math.Exp(-dist*dist*2.0)
				buf[fireH-1][x] = clamp(buf[fireH-1][x]+base+(rand.Float64()-0.5)*0.15, 0, 1)
			}

			// Sparks - small bright dots that fly up
			if rand.Intn(3) == 0 {
				sx := hearthLeft + rand.Intn(hearthW)
				sy := fireH - 2 - rand.Intn(fireH/4)
				if sy >= 0 && sy < fireH {
					sparks[sy][sx] = 0.4 + rand.Float64()*0.3
				}
			}

			// Propagate upward - key to flame shape
			for y := 0; y < fireH-1; y++ {
				for x := 0; x < fireW; x++ {
					// Tight vertical kernel - flames go UP, not sideways
					sum := buf[y+1][x] * 3.0
					count := 3.0
					if x > 0 {
						sum += buf[y+1][x-1] * 0.5
						count += 0.5
					}
					if x < fireW-1 {
						sum += buf[y+1][x+1] * 0.5
						count += 0.5
					}
					avg := sum / count

					// Gentle wind sway
					windDrift := math.Sin(t*0.05+float64(y)*0.3) * 0.8
					drift := int(math.Round(windDrift))
					dx := clampInt(x+drift, 0, fireW-1)
					avg = (avg*3 + buf[y+1][dx]) / 4.0

					// Cooling: gentle near base, aggressive near top
					heightRatio := float64(y) / float64(fireH) // 0 at top, 1 at bottom
					cool := 0.04 + (1.0-heightRatio)*0.05 + rand.Float64()*0.03
					// Random extra cooling creates ragged edges
					if rand.Intn(8) == 0 {
						cool += 0.1
					}
					buf[y][x] = clamp(avg-cool, 0, 1)

					// Sparks
					if sparks[y][x] > 0 {
						buf[y][x] = clamp(buf[y][x]+sparks[y][x], 0, 1)
						sparks[y][x] *= 0.4
						if sparks[y][x] < 0.03 {
							sparks[y][x] = 0
						}
						if y > 0 {
							sparks[y-1][x] = clamp(sparks[y-1][x]+sparks[y][x]*0.6, 0, 1)
						}
					}
				}
			}

			// Slowly drift tongue positions for variety
			if frame%10 == 0 {
				for i := range tongues {
					tongues[i].x += (rand.Float64() - 0.5) * 0.5
					tongues[i].x = clamp(tongues[i].x, float64(hearthLeft+1), float64(hearthRight-2))
					tongues[i].intensity = clamp(tongues[i].intensity+(rand.Float64()-0.5)*0.1, 0.3, 1.0)
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
