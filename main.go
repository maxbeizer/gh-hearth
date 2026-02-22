package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	width  = 40
	height = 18
)

// Lo-fi: just a few ASCII chars, hottest to coolest
var chars = []byte{'.', ':', '*', '^', '~', ' '}

func main() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	fmt.Print("\033[?25l\033[2J") // hide cursor, clear screen
	defer fmt.Print("\033[?25h\033[0m\n")

	// Heat buffer: 0.0 = cold, 1.0 = hot
	buf := make([][]float64, height)
	for i := range buf {
		buf[i] = make([]float64, width)
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sig:
			return
		case <-ticker.C:
			// Seed the bottom row
			for x := 0; x < width; x++ {
				// Hotter in the middle, cooler at edges
				center := float64(width) / 2
				dist := abs(float64(x) - center) / center
				base := 1.0 - dist*0.6
				jitter := (rand.Float64() - 0.5) * 0.4
				buf[height-1][x] = clamp(base+jitter, 0, 1)
			}

			// Propagate upward: each cell averages neighbors below + cools
			for y := 0; y < height-1; y++ {
				for x := 0; x < width; x++ {
					below := buf[y+1][x]
					left := below
					right := below
					if x > 0 {
						left = buf[y+1][x-1]
					}
					if x < width-1 {
						right = buf[y+1][x+1]
					}
					// Slight random horizontal drift
					drift := rand.Intn(3) - 1
					dx := clampInt(x+drift, 0, width-1)
					avg := (below + left + right + buf[y+1][dx]) / 4.0

					// Cool as it rises
					cool := 0.15 + rand.Float64()*0.08
					buf[y][x] = clamp(avg-cool, 0, 1)
				}
			}

			// Render
			var sb strings.Builder
			sb.WriteString("\033[H")

			// Some top padding
			sb.WriteString("\n\n")

			for y := 0; y < height; y++ {
				// Center the fire
				sb.WriteString("                    ")
				for x := 0; x < width; x++ {
					heat := buf[y][x]
					if heat < 0.05 {
						sb.WriteByte(' ')
					} else {
						idx := int((1.0 - heat) * float64(len(chars)-1))
						if idx < 0 {
							idx = 0
						}
						if idx >= len(chars) {
							idx = len(chars) - 1
						}
						sb.WriteByte(chars[idx])
					}
				}
				sb.WriteByte('\n')
			}

			// Logs / base
			sb.WriteString("                    ")
			sb.WriteString("________________________________________\n")
			sb.WriteString("                    ")
			sb.WriteString("   ===[]========[]====[]========[]===   \n")
			sb.WriteString("\n")
			sb.WriteString("                          gh hearth\n")

			fmt.Print(sb.String())
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
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
