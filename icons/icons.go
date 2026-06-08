package icons

import (
	"bytes"
	"encoding/binary"
	"image/color"
)

// Pre-generated icon bytes for each state.
var (
	Gray  []byte // proxy off / direct
	Green []byte // system proxy on, rule/global
	Blue  []byte // TUN mode
)

func init() {
	Gray = Generate(color.RGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xFF})
	Green = Generate(color.RGBA{R: 0x4C, G: 0xAF, B: 0x50, A: 0xFF})
	Blue = Generate(color.RGBA{R: 0x21, G: 0x96, B: 0xF3, A: 0xFF})
}

// Generate creates an ICO byte slice containing a solid-color square icon
// in three sizes: 16×16, 24×24, 32×32 pixels at 32bpp (BGRA).
func Generate(c color.RGBA) []byte {
	sizes := []int{16, 24, 32}

	// Pre-generate all DIB image data so we can compute offsets.
	type imgEntry struct {
		width  int
		height int
		data   []byte
	}
	images := make([]imgEntry, len(sizes))
	for i, sz := range sizes {
		images[i] = imgEntry{sz, sz, generateDIB(sz, sz, c)}
	}

	// Calculate file offsets.
	headerSize := 6 + 16*len(sizes) // ICO header + directory entries
	offsets := make([]int, len(sizes))
	running := headerSize
	for i := range sizes {
		offsets[i] = running
		running += len(images[i].data)
	}

	var buf bytes.Buffer

	// ICO header
	writeLE(&buf, uint16(0))           // reserved
	writeLE(&buf, uint16(1))           // type: ICO
	writeLE(&buf, uint16(len(sizes)))  // image count

	// Directory entries
	for i, img := range images {
		w, h := img.width, img.height
		if w >= 256 {
			w = 0
		}
		if h >= 256 {
			h = 0
		}
		buf.WriteByte(byte(w))
		buf.WriteByte(byte(h))
		buf.WriteByte(0)                     // color palette count (0 = no palette)
		buf.WriteByte(0)                     // reserved
		writeLE(&buf, uint16(1))             // color planes
		writeLE(&buf, uint16(32))            // bits per pixel
		writeLE(&buf, uint32(len(img.data))) // image size
		writeLE(&buf, uint32(offsets[i]))    // file offset
	}

	// Image data
	for _, img := range images {
		buf.Write(img.data)
	}

	return buf.Bytes()
}

// generateDIB creates a DIB (Device Independent Bitmap) for use inside an ICO.
// The returned data is: BITMAPINFOHEADER + pixel rows (bottom-up BGRA) + AND mask.
func generateDIB(width, height int, c color.RGBA) []byte {
	var buf bytes.Buffer

	// Pixel rows, bottom-up.
	rowSize := width * 4 // BGRA
	pixelData := make([]byte, 0, rowSize*height)
	for y := height - 1; y >= 0; y-- {
		for x := 0; x < width; x++ {
			pixelData = append(pixelData, c.B, c.G, c.R, 0xFF)
		}
	}

	// AND mask: 1 bit per pixel, each row padded to 4-byte boundary.
	andRowBytes := ((width + 31) / 32) * 4
	andMask := make([]byte, andRowBytes*height)
	// All zeros = fully opaque (no transparency).

	// BITMAPINFOHEADER (40 bytes)
	writeLE(&buf, uint32(40))                        // biSize
	writeLE(&buf, int32(width))                      // biWidth
	writeLE(&buf, int32(height*2))                   // biHeight (×2: image + AND mask)
	writeLE(&buf, uint16(1))                         // biPlanes
	writeLE(&buf, uint16(32))                        // biBitCount
	writeLE(&buf, uint32(0))                         // biCompression: BI_RGB
	writeLE(&buf, uint32(len(pixelData)))            // biSizeImage
	writeLE(&buf, int32(2835))                       // biXPelsPerMeter (~72 DPI)
	writeLE(&buf, int32(2835))                       // biYPelsPerMeter (~72 DPI)
	writeLE(&buf, uint32(0))                         // biClrUsed
	writeLE(&buf, uint32(0))                         // biClrImportant

	buf.Write(pixelData)
	buf.Write(andMask)

	return buf.Bytes()
}

// writeLE writes a value in little-endian byte order.
func writeLE(buf *bytes.Buffer, v interface{}) {
	_ = binary.Write(buf, binary.LittleEndian, v)
}
