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
)

const (
	width  = 60
	height = 25
)

// Fire palette from dark to bright
var palette = []string{
	"\033[38;2;20;0;0m",   // near-black red
	"\033[38;2;80;10;0m",  // dark red
	"\033[38;2;150;30;0m", // red
	"\033[38;2;200;60;0m", // orange-red
	"\033[38;2;220;120;0m",// orange
	"\033[38;2;240;180;20m",// yellow-orange
	"\033[38;2;255;220;80m",// yellow
	"\033[38;2;255;255;180m",// bright yellow
	"\033[38;2;255;255;255m",// white-hot
}

var chars = []rune{'░', '▒', '▓', '█', '▓', '▒', '░', ' '}

var reset = "\033[0m"

func main() {
	// Handle exit gracefully
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	// Hide cursor
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h\033[0m\n")

	// Fire buffer (bottom row is hottest)
	buf := make([][]float64, height)
	for i := range buf {
		buf[i] = make([]float64, width)
	}

	ticker := time.NewTicker(60 * time.Millisecond)
	defer ticker.Stop()

	frame := 0
	for {
		select {
		case <-sig:
			return
		case <-ticker.C:
			// Seed bottom rows with heat
			for x := 0; x < width; x++ {
				// Base heat with variation
				heat := 0.7 + rand.Float64()*0.3
				// Occasional flare-ups
				if rand.Float64() < 0.1 {
					heat = 1.0
				}
				// Cooler edges
				edgeDist := float64(min(x, width-1-x)) / float64(width/2)
				heat *= 0.4 + 0.6*edgeDist
				buf[height-1][x] = heat
				buf[height-2][x] = heat * (0.8 + rand.Float64()*0.2)
			}

			// Propagate fire upward with cooling and spread
			for y := 0; y < height-2; y++ {
				for x := 0; x < width; x++ {
					// Sample from below with random horizontal spread
					spread := rand.Intn(3) - 1
					sx := clamp(x+spread, 0, width-1)
					// Average of neighbors below
					heat := buf[y+1][sx]
					if x > 0 {
						heat += buf[y+1][x-1]
					} else {
						heat += buf[y+1][x]
					}
					if x < width-1 {
						heat += buf[y+1][x+1]
					} else {
						heat += buf[y+1][x]
					}
					heat += buf[y+2][sx]
					heat /= 4.0

					// Cool as it rises — more cooling near top
					coolFactor := 0.94 - float64(height-1-y)*0.003
					// Add flicker
					flicker := 1.0 + (rand.Float64()-0.5)*0.1
					heat *= coolFactor * flicker

					// Sine wave for organic movement
					wave := math.Sin(float64(frame)*0.05+float64(x)*0.15) * 0.03
					heat += wave

					buf[y][x] = clamp64(heat, 0, 1)
				}
			}

			// Render
			var sb strings.Builder
			sb.WriteString("\033[H") // Move cursor to top-left

			for y := 0; y < height; y++ {
				for x := 0; x < width; x++ {
					heat := buf[y][x]
					if heat < 0.05 {
						sb.WriteRune(' ')
						continue
					}
					// Map heat to palette
					idx := int(heat * float64(len(palette)-1))
					idx = clamp(idx, 0, len(palette)-1)
					// Map heat to char
					cidx := int((1.0 - heat) * float64(len(chars)-1))
					cidx = clamp(cidx, 0, len(chars)-1)

					sb.WriteString(palette[idx])
					sb.WriteRune(chars[cidx])
				}
				sb.WriteString(reset)
				sb.WriteRune('\n')
			}

			// Fireplace base
			sb.WriteString("\033[38;2;100;80;60m")
			sb.WriteString(strings.Repeat("━", width))
			sb.WriteString(reset + "\n")

			// Branding
			sb.WriteString("\033[38;2;180;140;100m")
			pad := (width - 22) / 2
			sb.WriteString(strings.Repeat(" ", pad))
			sb.WriteString("🔥  gh hearth  v0.1.0")
			sb.WriteString(reset)

			fmt.Print(sb.String())
			frame++
		}
	}
}

func clamp(v, lo, hi int) int {
	if v < lo { return lo }
	if v > hi { return hi }
	return v
}

func clamp64(v, lo, hi float64) float64 {
	if v < lo { return lo }
	if v > hi { return hi }
	return v
}

func min(a, b int) int {
	if a < b { return a }
	return b
}
