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

	// Flame tongues: more than final (denser) but fewer than before
	numTongues := hearthW / 6
	if numTongues < 4 {
		numTongues = 4
	}
	if numTongues > 15 {
		numTongues = 15
	}
	tongues := make([]tongue, numTongues)
	spacing := float64(hearthW) / float64(numTongues+1)
	for i := range tongues {
		tongues[i] = tongue{
			x:         float64(hearthLeft) + spacing*float64(i+1) + (rand.Float64()-0.5)*spacing*0.3,
			intensity: 0.55 + rand.Float64()*0.45,
			speed:     0.07 + rand.Float64()*0.11,
			phase:     rand.Float64() * math.Pi * 2,
			width:     1.2 + rand.Float64()*2.0,
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
					numTongues = hearthW / 6
					if numTongues < 4 {
						numTongues = 4
					}
					if numTongues > 15 {
						numTongues = 15
					}
					tongues = make([]tongue, numTongues)
					spacing := float64(hearthW) / float64(numTongues+1)
					for i := range tongues {
						tongues[i] = tongue{
							x:         float64(hearthLeft) + spacing*float64(i+1) + (rand.Float64()-0.5)*spacing*0.3,
							intensity: 0.55 + rand.Float64()*0.45,
							speed:     0.07 + rand.Float64()*0.11,
							phase:     rand.Float64() * math.Pi * 2,
							width:     1.2 + rand.Float64()*2.0,
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

			// Each tongue is a gaussian heat source
			t := float64(frame)
			for _, tng := range tongues {
				sway := math.Sin(t*tng.speed+tng.phase) * 2.5
				cx := tng.x + sway
				pulse := tng.intensity * (0.6 + 0.4*math.Sin(t*tng.speed*2.0+tng.phase))

				for x := hearthLeft; x < hearthRight; x++ {
					dist := (float64(x) - cx) / tng.width
					heat := pulse * math.Exp(-dist*dist*0.7)
					if heat > 0.05 {
						buf[fireH-1][x] = clamp(buf[fireH-1][x]+heat, 0, 1)
					}
				}
			}

			// Moderate baseline warmth
			for x := hearthLeft; x < hearthRight; x++ {
				dist := math.Abs(float64(x)-float64(hearthLeft+hearthW/2)) / float64(hearthW/2)
				base := 0.2 * math.Exp(-dist*dist*1.8)
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
					if rand.Intn(10) == 0 {
						cool += 0.12
					}

					val := clamp(avg-cool, 0, 1)

					// Cull low-heat pixels for wispy tips
					if val < 0.12 && rand.Intn(3) != 0 {
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

			// Hearth base: logs with glowing embers
			pad := (fireW - hearthW) / 2

			// Dense hot coals right at fire base
			sb.WriteString(strings.Repeat(" ", pad))
			coalChars := []byte{'#', '@', '%', '&', '*', '#', '@', '%'}
			for i := 0; i < hearthW; i++ {
				sb.WriteByte(coalChars[rand.Intn(len(coalChars))])
			}
			sb.WriteByte('\n')

			// Logs: two crossed logs made of ()= with ember gaps
			sb.WriteString(strings.Repeat(" ", pad))
			for i := 0; i < hearthW; i++ {
				pos := float64(i) / float64(hearthW)
				// Two log shapes crossing
				onLog1 := math.Abs(pos-0.3) < 0.25
				onLog2 := math.Abs(pos-0.7) < 0.25
				if onLog1 || onLog2 {
					if rand.Intn(6) == 0 {
						// Ember glow in log cracks
						ec := []byte{'*', ':', '+'}
						sb.WriteByte(ec[rand.Intn(len(ec))])
					} else {
						lc := []byte{'(', ')', '=', '=', '0', 'O'}
						sb.WriteByte(lc[rand.Intn(len(lc))])
					}
				} else {
					// Embers between logs
					ec := []byte{'.', ',', ':', ' ', ' '}
					sb.WriteByte(ec[rand.Intn(len(ec))])
				}
			}
			sb.WriteByte('\n')

			// Stone hearth floor
			sb.WriteString(strings.Repeat(" ", pad))
			for i := 0; i < hearthW; i++ {
				if i%(8+rand.Intn(4)) == 0 {
					sb.WriteByte('|')
				} else {
					sb.WriteByte('_')
				}
			}
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
