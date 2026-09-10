package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"

	"github.com/fyne-io/oksvg"
	"github.com/srwiley/rasterx"
)

var icoSizes = []int{16, 24, 32, 48, 64, 128, 256}
var pngSizes = []int{16, 24, 32, 48}

func render(svgPath string, size int) (*image.RGBA, error) {
	icon, err := oksvg.ReadIcon(svgPath, oksvg.WarnErrorMode)
	if err != nil {
		return nil, err
	}
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, img, img.Bounds())
	raster := rasterx.NewDasher(size, size, scanner)
	icon.SetTarget(0, 0, float64(size), float64(size))
	icon.Draw(raster, 1.0)
	return img, nil
}

// writeICO packs PNG-compressed entries into a multi-size .ico (Vista+ format),
// used as the executable's resource icon.
func writeICO(path, svgPath string, sizes []int) error {
	type entry struct {
		dim  byte
		data []byte
	}
	var entries []entry
	for _, size := range sizes {
		img, err := render(svgPath, size)
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return err
		}
		d := byte(size)
		if size >= 256 {
			d = 0
		}
		entries = append(entries, entry{dim: d, data: buf.Bytes()})
	}

	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, uint16(0))
	binary.Write(&out, binary.LittleEndian, uint16(1))
	binary.Write(&out, binary.LittleEndian, uint16(len(entries)))

	offset := uint32(6 + 16*len(entries))
	for _, e := range entries {
		out.WriteByte(e.dim)
		out.WriteByte(e.dim)
		out.WriteByte(0)
		out.WriteByte(0)
		binary.Write(&out, binary.LittleEndian, uint16(1))
		binary.Write(&out, binary.LittleEndian, uint16(32))
		binary.Write(&out, binary.LittleEndian, uint32(len(e.data)))
		binary.Write(&out, binary.LittleEndian, offset)
		offset += uint32(len(e.data))
	}
	for _, e := range entries {
		out.Write(e.data)
	}
	return os.WriteFile(path, out.Bytes(), 0o644)
}

// writePNGs emits one bitmap per size so the window icon can be handed to
// Windows at the exact pixel size it asks for instead of being downscaled.
func writePNGs(dir, svgPath string, sizes []int) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, size := range sizes {
		img, err := render(svgPath, size)
		if err != nil {
			return err
		}
		name := filepath.Join(dir, "icon-"+strconv.Itoa(size)+".png")
		f, err := os.Create(name)
		if err != nil {
			return err
		}
		if err := png.Encode(f, img); err != nil {
			f.Close()
			return err
		}
		f.Close()
		fmt.Println("wrote", name)
	}
	return nil
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("usage: svgrast <file.svg> <ico-out> <png-dir>")
		os.Exit(2)
	}
	svgPath := os.Args[1]

	if err := writeICO(os.Args[2], svgPath, icoSizes); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %s (%d sizes)\n", os.Args[2], len(icoSizes))

	if err := writePNGs(os.Args[3], svgPath, pngSizes); err != nil {
		panic(err)
	}
}
