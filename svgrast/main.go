package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/fyne-io/oksvg"
	"github.com/srwiley/rasterx"
)

var (
	icoSizes = []int{16, 24, 32, 48, 64, 128, 256}

	// assetsDir 是图标源稿目录；baseName 用于没有手调尺寸时的大图。
	assetsDir = "assets"
	baseName  = "appicon.svg"

	strokeRe  = regexp.MustCompile(`stroke-width="([0-9.]+)"`)
	viewBoxRe = regexp.MustCompile(`viewBox="[0-9.\-]+ [0-9.\-]+ ([0-9.]+) ([0-9.]+)"`)
)

// sourceFor 返回某个尺寸应当使用的源稿：优先 appicon-<size>.svg（按像素手调，
// 描边正好落在像素边界上），否则回退到基准稿。
func sourceFor(size int) (string, bool) {
	tuned := filepath.Join(assetsDir, fmt.Sprintf("appicon-%d.svg", size))
	if _, err := os.Stat(tuned); err == nil {
		return tuned, true
	}
	return filepath.Join(assetsDir, baseName), false
}

// render 把 SVG 栅格化成 size×size。
//
// oksvg 把 stroke-width 当作**设备像素**处理，不随 viewBox→target 的缩放而变化
// （svg_path.go 里是 r.SetStroke(fixed.Int26_6(svgp.LineWidth*64))，而 SetTarget 只
// 改变换矩阵）。所以把 48 单位的基准稿放大到 256 时，6 单位的描边只会画成 6px 细线。
// 这里在解析前按同样的比例换算描边宽度，让各尺寸的视觉一致。
func render(svgPath string, size int) (*image.RGBA, error) {
	raw, err := os.ReadFile(svgPath)
	if err != nil {
		return nil, err
	}
	if m := viewBoxRe.FindSubmatch(raw); m != nil {
		if view, err := strconv.ParseFloat(string(m[1]), 64); err == nil && view > 0 && view != float64(size) {
			raw = scaleStrokes(raw, float64(size)/view)
		}
	}

	icon, err := oksvg.ReadIconStream(bytes.NewReader(raw), oksvg.WarnErrorMode)
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

// scaleStrokes 按 scale 放大所有 stroke-width。
func scaleStrokes(raw []byte, scale float64) []byte {
	return strokeRe.ReplaceAllFunc(raw, func(m []byte) []byte {
		g := strokeRe.FindSubmatch(m)
		w, err := strconv.ParseFloat(string(g[1]), 64)
		if err != nil {
			return m
		}
		return []byte(`stroke-width="` + strconv.FormatFloat(w*scale, 'g', -1, 64) + `"`)
	})
}

// writeICO packs one BMP-encoded entry per size into a classic multi-size .ico,
// used both as the executable's resource icon and as the source of the window icon.
//
// 条目必须是 BMP 而不是 PNG：go-winres 会把 ICO 条目原样搬进 RT_ICON 资源，而
// LoadImage 不认资源里的 PNG 图标（实测 16/32/48/64/256 全部返回 NULL），
// 程序就取不到自己的窗口图标。
func writeICO(path string, sizes []int) error {
	type entry struct {
		dim  byte
		data []byte
	}
	var entries []entry
	for _, size := range sizes {
		src, _ := sourceFor(size)
		img, err := render(src, size)
		if err != nil {
			return err
		}
		d := byte(size)
		if size >= 256 {
			d = 0
		}
		entries = append(entries, entry{dim: d, data: iconBMP(img, size)})
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

// iconBMP 把位图编码成 ICO 的经典 BMP 条目：BITMAPINFOHEADER（biHeight 为 2*size，
// 下半段留给 AND 掩码）+ 自下而上的 BGRA 像素 + 全 0 掩码。
//
// 像素要还原成**非预乘** alpha：rasterx 写进 image.RGBA 的是预乘值，而图标位图约定
// 用直通 alpha，否则半透明的抗锯齿边缘会被压暗。
func iconBMP(img *image.RGBA, size int) []byte {
	maskStride := ((size + 31) / 32) * 4
	pixels := size * size * 4
	mask := maskStride * size

	out := make([]byte, 0, 40+pixels+mask)
	var hdr [40]byte
	binary.LittleEndian.PutUint32(hdr[0:], 40)
	binary.LittleEndian.PutUint32(hdr[4:], uint32(size))
	binary.LittleEndian.PutUint32(hdr[8:], uint32(size*2))
	binary.LittleEndian.PutUint16(hdr[12:], 1)
	binary.LittleEndian.PutUint16(hdr[14:], 32)
	binary.LittleEndian.PutUint32(hdr[20:], uint32(pixels+mask))
	out = append(out, hdr[:]...)

	row := make([]byte, 4)
	for y := size - 1; y >= 0; y-- {
		for x := 0; x < size; x++ {
			i := y*img.Stride + x*4
			a := img.Pix[i+3]
			row[3] = a
			if a == 0 {
				row[0], row[1], row[2] = 0, 0, 0
			} else {
				row[0] = unpremultiply(img.Pix[i+2], a) // B
				row[1] = unpremultiply(img.Pix[i+1], a) // G
				row[2] = unpremultiply(img.Pix[i+0], a) // R
			}
			out = append(out, row...)
		}
	}
	// 单色 AND 掩码全 0：配合 alpha 通道即完整的分层图标。
	return append(out, make([]byte, mask)...)
}

func unpremultiply(v, a byte) byte {
	if a >= 255 {
		return v
	}
	w := int(v) * 255 / int(a)
	if w > 255 {
		return 255
	}
	return byte(w)
}

// preview prints a rendered size as ASCII so the glyph can be judged 1:1.
func preview(size int) {
	src, tuned := sourceFor(size)
	img, err := render(src, size)
	if err != nil {
		panic(err)
	}
	kind := "scaled"
	if tuned {
		kind = "hand-tuned"
	}
	fmt.Printf("===== %dpx (%s: %s) =====\n", size, kind, src)
	for y := 0; y < size; y++ {
		line := make([]byte, size)
		for x := 0; x < size; x++ {
			r, _, _, a := img.At(x, y).RGBA()
			switch {
			case a>>8 == 0:
				line[x] = ' '
			case r>>8 > 128:
				line[x] = '#'
			default:
				line[x] = ' '
			}
		}
		fmt.Println(string(line))
	}
}

func usage() {
	fmt.Println("usage: svgrast <ico-out> | svgrast --preview <size>...")
	os.Exit(2)
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
	}

	if args[0] == "--preview" {
		for _, a := range args[1:] {
			n, err := strconv.Atoi(a)
			if err != nil {
				panic(err)
			}
			preview(n)
		}
		return
	}

	if len(args) < 1 {
		usage()
	}
	if err := writeICO(args[0], icoSizes); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %s (%d sizes)\n", args[0], len(icoSizes))
}
