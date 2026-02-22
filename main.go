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

// Graduated from dense/hot to wispy/cool
var fireChars = []byte{'#', '@', '%', '&', '*', '+', ':', '~', '-', '.', '`'}

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

	// Create fewer, well-spaced flame tongues
	numTongues := hearthW / 8
	if numTongues < 3 {
		numTongues = 3
	}
	if numTongues > 12 {
		numTongues = 12
	}
	tongues := make([]tongue, numTongues)
	spacing := float64(hearthW) / float64(numTongues+1)
	for i := range tongues {
		tongues[i] = tongue{
			x:         float64(hearthLeft) + spacing*float64(i+1) + (rand.Float64()-0.5)*spacing*0.4,
			intensity: 0.6 + rand.Float64()*0.4,
			speed:     0.06 + rand.Float64()*0.1,
			phase:     rand.Float64() * math.Pi * 2,
			width:     1.0 + rand.Float64()*1.5,
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
					numTongues = hearthW / 8
					if numTongues < 3 {
						numTongues = 3
					}
					if numTongues > 12 {
						numTongues = 12
					}
					tongues = make([]tongue, numTongues)
					spacing := float64(hearthW) / float64(numTongues+1)
					for i := range tongues {
						tongues[i] = tongue{
							x:         float64(hearthLeft) + spacing*float64(i+1) + (rand.Float64()-0.5)*spacing*0.4,
							intensity: 0.6 + rand.Float64()*0.4,
							speed:     0.06 + rand.Float64()*0.1,
							phase:     rand.Float64() * math.Pi * 2,
							width:     1.0 + rand.Float64()*1.5,
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

			// Each tongue is a narrow gaussian heat source
			t := float64(frame)
			for _, tng := range tongues {
				sway := math.Sin(t*tng.speed+tng.phase) * 3.0
				cx := tng.x + sway
				pulse := tng.intensity * (0.5 + 0.5*math.Sin(t*tng.speed*2.3+tng.phase))

				for x := hearthLeft; x < hearthRight; x++ {
					dist := (float64(x) - cx) / tng.width
					heat := pulse * math.Exp(-dist*dist*0.5)
					if heat > 0.05 {
						buf[fireH-1][x] = clamp(buf[fireH-1][x]+heat, 0, 1)
					}
				}
			}

			// Light baseline warmth - just enough to connect at base
			for x := hearthLeft; x < hearthRight; x++ {
				dist := math.Abs(float64(x)-float64(hearthLeft+hearthW/2)) / float64(hearthW/2)
				base := 0.15 * math.Exp(-dist*dist*1.5)
				buf[fireH-1][x] = clamp(buf[fireH-1][x]+base, 0, 1)
			}

			// Sparks - small bright dots that fly up
			if rand.Intn(3) == 0 {
				sx := hearthLeft + rand.Intn(hearthW)
				sy := fireH - 2 - rand.Intn(fireH/4)
				if sy >= 0 && sy < fireH {
					sparks[sy][sx] = 0.4 + rand.Float64()*0.3
				}
			}

			// Propagate upward - strongly vertical
			for y := 0; y < fireH-1; y++ {
				for x := 0; x < fireW; x++ {
					// Heavy center weight = flames go straight up
					sum := buf[y+1][x] * 5.0
					count := 5.0
					if x > 0 {
						sum += buf[y+1][x-1] * 0.3
						count += 0.3
					}
					if x < fireW-1 {
						sum += buf[y+1][x+1] * 0.3
						count += 0.3
					}
					avg := sum / count

					// Gentle sway
					windDrift := math.Sin(t*0.04+float64(y)*0.25) * 0.6
					drift := int(math.Round(windDrift))
					if drift != 0 {
						dx := clampInt(x+drift, 0, fireW-1)
						avg = (avg*4 + buf[y+1][dx]) / 5.0
					}

					// Cooling
					heightRatio := float64(y) / float64(fireH)
					cool := 0.04 + (1.0-heightRatio)*0.04 + rand.Float64()*0.025

					// Random extinction for ragged edges
					if rand.Intn(12) == 0 {
						cool += 0.15
					}

					val := clamp(avg-cool, 0, 1)

					// Cull low-heat pixels for sparse wispy tips
					if val < 0.15 && rand.Intn(3) != 0 {
						val = 0
					}

					buf[y][x] = val

					// Sparks
					if sparks[y][x] > 0 {
						buf[y][x] = clamp(buf[y][x]+sparks[y][x], 0, 1)
						sparks[y][x] *= 0.35
						if sparks[y][x] < 0.03 {
							sparks[y][x] = 0
						}
						if y > 0 {
							sparks[y-1][x] = clamp(sparks[y-1][x]+sparks[y][x]*0.5, 0, 1)
						}
					}
				}
			}

			// Drift tongue positions slowly
			if frame%15 == 0 {
				for i := range tongues {
					tongues[i].x += (rand.Float64() - 0.5) * 0.8
					tongues[i].x = clamp(tongues[i].x, float64(hearthLeft+2), float64(hearthRight-3))
					tongues[i].intensity = clamp(tongues[i].intensity+(rand.Float64()-0.5)*0.08, 0.4, 1.0)
					tongues[i].width = clamp(tongues[i].width+(rand.Float64()-0.5)*0.2, 0.8, 2.5)
				}
			}

			// Render
			var sb strings.Builder
			sb.Grow(fireW * fireH * 2)
			sb.WriteString("\033[H")

			for y := 0; y < fireH; y++ {
				for x := 0; x < fireW; x++ {
					heat := buf[y][x]
					if heat < 0.03 {
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
