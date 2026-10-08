// Command genicon renders the LocalMail application icon (build/appicon.png)
// from vector shapes using signed distance fields, so it is reproducible and
// needs no design tools. Run: go run ./scripts/genicon
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
)

const size = 1024

func main() {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	top := [3]float64{99, 91, 255} // #635BFF
	bot := [3]float64{67, 56, 202} // #4338CA
	white := [3]float64{255, 255, 255}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5

			// Background: rounded square with a subtle vertical gradient.
			bg := coverage(sdRoundRect(px, py, 512, 512, 440, 440, 200))
			t := py / size
			col := [3]float64{lerp(top[0], bot[0], t), lerp(top[1], bot[1], t), lerp(top[2], bot[2], t)}

			// Envelope: outlined rounded rectangle plus the flap "V".
			const stroke = 30.0
			body := math.Abs(sdRoundRect(px, py, 512, 540, 270, 190, 48)) - stroke
			flapL := sdSegment(px, py, 262, 380, 512, 560) - stroke
			flapR := sdSegment(px, py, 762, 380, 512, 560) - stroke
			env := coverage(math.Min(body, math.Min(flapL, flapR)))

			// Notification dot (new mail) at the top-right of the envelope.
			dotRing := coverage(sdCircle(px, py, 760, 340, 92))
			dot := coverage(sdCircle(px, py, 760, 340, 62))

			c := col
			c = mix(c, white, env)
			c = mix(c, col, dotRing) // cut-out ring separating dot from envelope
			c = mix(c, [3]float64{52, 211, 153}, dot)

			img.SetNRGBA(x, y, color.NRGBA{uint8(c[0]), uint8(c[1]), uint8(c[2]), uint8(255 * bg)})
		}
	}

	f, err := os.Create("build/appicon.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
	log.Println("wrote build/appicon.png")
}

func coverage(d float64) float64 { return clamp(0.5-d, 0, 1) }

func sdRoundRect(px, py, cx, cy, hw, hh, r float64) float64 {
	qx := math.Abs(px-cx) - hw + r
	qy := math.Abs(py-cy) - hh + r
	ox, oy := math.Max(qx, 0), math.Max(qy, 0)
	return math.Hypot(ox, oy) + math.Min(math.Max(qx, qy), 0) - r
}

func sdSegment(px, py, ax, ay, bx, by float64) float64 {
	pax, pay := px-ax, py-ay
	bax, bay := bx-ax, by-ay
	h := clamp((pax*bax+pay*bay)/(bax*bax+bay*bay), 0, 1)
	return math.Hypot(pax-bax*h, pay-bay*h)
}

func sdCircle(px, py, cx, cy, r float64) float64 { return math.Hypot(px-cx, py-cy) - r }

func mix(a, b [3]float64, t float64) [3]float64 {
	return [3]float64{lerp(a[0], b[0], t), lerp(a[1], b[1], t), lerp(a[2], b[2], t)}
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
