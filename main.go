package main

import (
	"flag"
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

// Color palette: white-hot -> yellow -> orange -> red -> dark red
var palette = []string{
	"\033[38;2;255;255;255m", // white-hot core
	"\033[38;2;255;255;180m", // bright yellow-white
	"\033[38;2;255;240;80m",  // yellow
	"\033[38;2;255;200;30m",  // golden
	"\033[38;2;255;160;0m",   // amber
	"\033[38;2;255;120;0m",   // orange
	"\033[38;2;240;70;0m",    // deep orange
	"\033[38;2;210;30;0m",    // red-orange
	"\033[38;2;180;15;0m",    // red
	"\033[38;2;130;5;0m",     // dark red
	"\033[38;2;80;2;0m",      // ember
}

var brickColor = "\033[38;2;140;70;30m"
var mantelColor = "\033[38;2;100;55;25m"
var coalColor = "\033[38;2;200;80;10m"
var logColor = "\033[38;2;120;60;20m"
var reset = "\033[0m"

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
	colorFlag := flag.Bool("color", false, "enable true-color flames")
	flag.Parse()
	useColor := *colorFlag

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	termW, termH := termSize()
	fireW := termW
	// Fireplace opening dimensions
	fpW := termW * 2 / 3
	if fpW > 80 {
		fpW = 80
	}
	fpLeft := (termW - fpW) / 2
	// Wall thickness
	wallW := 4
	// Fire zone: cap height so mantel sits just above flames
	maxFireH := 20
	fireH := termH - 8
	if fireH < 8 {
		fireH = 8
	}
	if fireH > maxFireH {
		fireH = maxFireH
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

	hearthW := fpW - wallW*2
	hearthLeft := fpLeft + wallW

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
					fpW = termW * 2 / 3
					if fpW > 80 {
						fpW = 80
					}
					fpLeft = (termW - fpW) / 2
					fireH = termH - 8
					if fireH < 8 {
						fireH = 8
					}
					if fireH > maxFireH {
						fireH = maxFireH
					}
					hearthW = fpW - wallW*2
					hearthLeft = fpLeft + wallW
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
			sb.Grow(fireW * (fireH + 10) * 6)
			sb.WriteString("\033[H")

			// Vertical centering: pad top if fireplace is shorter than terminal
			totalH := 2 + fireH + 4 // mantel(2) + fire + coals + logs(2) + floor
			topPad := (termH - totalH) / 2
			if topPad < 0 {
				topPad = 0
			}
			for i := 0; i < topPad; i++ {
				sb.WriteString(strings.Repeat(" ", termW))
				sb.WriteByte('\n')
			}

			// Mantel
			sb.WriteString(strings.Repeat(" ", fpLeft))
			if useColor {
				sb.WriteString(mantelColor)
			}
			sb.WriteByte(' ')
			sb.WriteString(strings.Repeat("_", fpW-2))
			sb.WriteByte(' ')
			if useColor {
				sb.WriteString(reset)
			}
			sb.WriteByte('\n')

			sb.WriteString(strings.Repeat(" ", fpLeft))
			if useColor {
				sb.WriteString(mantelColor)
			}
			sb.WriteByte('|')
			sb.WriteString(strings.Repeat("_", fpW-2))
			sb.WriteByte('|')
			if useColor {
				sb.WriteString(reset)
			}
			sb.WriteByte('\n')

			// Fire rows with brick walls
			for y := 0; y < fireH; y++ {
				sb.WriteString(strings.Repeat(" ", fpLeft))
				// Left wall
				if useColor {
					sb.WriteString(brickColor)
				}
				if y%2 == 0 {
					sb.WriteString("|__|")
				} else {
					sb.WriteString("|_|_")
				}
				if useColor {
					sb.WriteString(reset)
				}
				// Fire area
				for x := hearthLeft; x < hearthLeft+hearthW; x++ {
					if x >= 0 && x < fireW {
						heat := buf[y][x]
						if heat < 0.03 {
							if useColor {
								sb.WriteString(reset)
							}
							sb.WriteByte(' ')
							continue
						}
						fi := int((1.0 - heat) * float64(len(fireChars)-1))
						fi = clampInt(fi, 0, len(fireChars)-1)
						if useColor {
							ci := int((1.0 - heat) * float64(len(palette)-1))
							ci = clampInt(ci, 0, len(palette)-1)
							sb.WriteString(palette[ci])
						}
						sb.WriteByte(fireChars[fi])
					}
				}
				// Right wall
				if useColor {
					sb.WriteString(brickColor)
				}
				if y%2 == 0 {
					sb.WriteString("|__|")
				} else {
					sb.WriteString("_|_|")
				}
				if useColor {
					sb.WriteString(reset)
				}
				sb.WriteByte('\n')
			}

			// Coals row with walls
			sb.WriteString(strings.Repeat(" ", fpLeft))
			if useColor {
				sb.WriteString(brickColor)
			}
			if fireH%2 == 0 {
				sb.WriteString("|__|")
			} else {
				sb.WriteString("|_|_")
			}
			if useColor {
				sb.WriteString(coalColor)
			}
			coalChars := []byte{'#', '@', '%', '&', '*', '#', '@', '%'}
			for i := 0; i < hearthW; i++ {
				sb.WriteByte(coalChars[rand.Intn(len(coalChars))])
			}
			if useColor {
				sb.WriteString(brickColor)
			}
			if fireH%2 == 0 {
				sb.WriteString("|__|")
			} else {
				sb.WriteString("_|_|")
			}
			if useColor {
				sb.WriteString(reset)
			}
			sb.WriteByte('\n')

			// Top log row with walls
			sb.WriteString(strings.Repeat(" ", fpLeft))
			if useColor {
				sb.WriteString(brickColor)
			}
			sb.WriteString("|__|")
			if useColor {
				sb.WriteString(logColor)
			}
			logW := hearthW/4 - 2
			if logW < 4 {
				logW = 4
			}
			logGap := (hearthW - logW*3 - 6) / 4
			if logGap < 1 {
				logGap = 1
			}
			row := make([]byte, hearthW)
			for i := range row {
				row[i] = ' '
			}
			logStarts := []int{logGap, logGap*2 + logW + 2, logGap*3 + logW*2 + 4}
			for _, s := range logStarts {
				if s >= 0 && s < hearthW {
					row[s] = '('
				}
				for j := 1; j <= logW; j++ {
					if p := s + j; p >= 0 && p < hearthW {
						if rand.Intn(7) == 0 {
							row[p] = '*'
						} else {
							row[p] = '='
						}
					}
				}
				if e := s + logW + 1; e >= 0 && e < hearthW {
					row[e] = ')'
				}
			}
			sb.Write(row)
			if useColor {
				sb.WriteString(brickColor)
			}
			sb.WriteString("|__|")
			if useColor {
				sb.WriteString(reset)
			}
			sb.WriteByte('\n')

			// Bottom log row with walls
			sb.WriteString(strings.Repeat(" ", fpLeft))
			if useColor {
				sb.WriteString(brickColor)
			}
			sb.WriteString("|_|_")
			if useColor {
				sb.WriteString(logColor)
			}
			log2W := hearthW/3 - 2
			if log2W < 5 {
				log2W = 5
			}
			log2Gap := (hearthW - log2W*2 - 4) / 3
			if log2Gap < 1 {
				log2Gap = 1
			}
			row2 := make([]byte, hearthW)
			for i := range row2 {
				row2[i] = ' '
			}
			log2Starts := []int{log2Gap, log2Gap*2 + log2W + 2}
			for _, s := range log2Starts {
				if s >= 0 && s < hearthW {
					row2[s] = '('
				}
				for j := 1; j <= log2W; j++ {
					if p := s + j; p >= 0 && p < hearthW {
						if rand.Intn(8) == 0 {
							row2[p] = '*'
						} else {
							row2[p] = '='
						}
					}
				}
				if e := s + log2W + 1; e >= 0 && e < hearthW {
					row2[e] = ')'
				}
			}
			sb.Write(row2)
			if useColor {
				sb.WriteString(brickColor)
			}
			sb.WriteString("_|_|")
			if useColor {
				sb.WriteString(reset)
			}
			sb.WriteByte('\n')

			// Hearth floor
			sb.WriteString(strings.Repeat(" ", fpLeft))
			if useColor {
				sb.WriteString(mantelColor)
			}
			sb.WriteByte('|')
			sb.WriteString(strings.Repeat("_", fpW-2))
			sb.WriteByte('|')
			if useColor {
				sb.WriteString(reset)
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
