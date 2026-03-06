package export

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"

	"knobman/internal/model"
	"knobman/internal/render"
)

var pngSignature = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}

type pngChunk struct {
	Type string
	Data []byte
}

// ExportAPNG renders the document as APNG bytes.
func ExportAPNG(doc *model.Document, textures []*render.Texture) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("export: nil document")
	}

	basePNGs, err := ExportPNGFrames(doc, textures)
	if err != nil {
		return nil, err
	}
	if len(basePNGs) == 0 {
		return nil, fmt.Errorf("export: no frames rendered")
	}

	sequence := gifFrameSequence(len(basePNGs), doc.Prefs.BiDir.Val != 0)
	if len(sequence) == 0 {
		return nil, fmt.Errorf("export: empty frame sequence")
	}

	firstChunks, err := parsePNGChunks(basePNGs[sequence[0]])
	if err != nil {
		return nil, fmt.Errorf("export: parse first png: %w", err)
	}
	ihdr := firstChunkData(firstChunks, "IHDR")
	if len(ihdr) != 13 {
		return nil, fmt.Errorf("export: missing IHDR")
	}
	width := binary.BigEndian.Uint32(ihdr[0:4])
	height := binary.BigEndian.Uint32(ihdr[4:8])
	if width == 0 || height == 0 {
		return nil, fmt.Errorf("export: invalid frame size %dx%d", width, height)
	}
	firstIDAT := allChunkData(firstChunks, "IDAT")
	if len(firstIDAT) == 0 {
		return nil, fmt.Errorf("export: first frame missing IDAT")
	}

	delayNum, delayDen := apngDelay(doc.Prefs.Duration.Val)
	numPlays := uint32(loopCountForGIF(doc.Prefs.Loop.Val))

	var out bytes.Buffer
	out.Write(pngSignature)
	writePNGChunk(&out, "IHDR", ihdr)

	acTL := make([]byte, 8)
	binary.BigEndian.PutUint32(acTL[0:4], uint32(len(sequence)))
	binary.BigEndian.PutUint32(acTL[4:8], numPlays)
	writePNGChunk(&out, "acTL", acTL)

	seqNo := uint32(0)
	writePNGChunk(&out, "fcTL", buildFCTL(seqNo, width, height, delayNum, delayDen))
	seqNo++
	for _, idat := range firstIDAT {
		writePNGChunk(&out, "IDAT", idat)
	}

	for i := 1; i < len(sequence); i++ {
		pngBytes := basePNGs[sequence[i]]
		chunks, err := parsePNGChunks(pngBytes)
		if err != nil {
			return nil, fmt.Errorf("export: parse frame %d png: %w", i, err)
		}
		idats := allChunkData(chunks, "IDAT")
		if len(idats) == 0 {
			return nil, fmt.Errorf("export: frame %d missing IDAT", i)
		}

		writePNGChunk(&out, "fcTL", buildFCTL(seqNo, width, height, delayNum, delayDen))
		seqNo++
		for _, idat := range idats {
			fdData := make([]byte, 4+len(idat))
			binary.BigEndian.PutUint32(fdData[0:4], seqNo)
			copy(fdData[4:], idat)
			writePNGChunk(&out, "fdAT", fdData)
			seqNo++
		}
	}

	writePNGChunk(&out, "IEND", nil)
	return out.Bytes(), nil
}

func apngDelay(durationMs int) (uint16, uint16) {
	ms := maxInt(1, durationMs)
	if ms > 65535 {
		ms = 65535
	}
	return uint16(ms), 1000
}

func buildFCTL(seqNo, width, height uint32, delayNum, delayDen uint16) []byte {
	data := make([]byte, 26)
	binary.BigEndian.PutUint32(data[0:4], seqNo)
	binary.BigEndian.PutUint32(data[4:8], width)
	binary.BigEndian.PutUint32(data[8:12], height)
	binary.BigEndian.PutUint32(data[12:16], 0) // x_offset
	binary.BigEndian.PutUint32(data[16:20], 0) // y_offset
	binary.BigEndian.PutUint16(data[20:22], delayNum)
	binary.BigEndian.PutUint16(data[22:24], delayDen)
	data[24] = 0 // dispose_op: none
	data[25] = 0 // blend_op: source
	return data
}

func parsePNGChunks(pngBytes []byte) ([]pngChunk, error) {
	if len(pngBytes) < len(pngSignature) || !bytes.Equal(pngBytes[:len(pngSignature)], pngSignature) {
		return nil, fmt.Errorf("invalid png signature")
	}
	pos := len(pngSignature)
	out := make([]pngChunk, 0, 16)
	for {
		if pos+12 > len(pngBytes) {
			return nil, fmt.Errorf("truncated chunk header")
		}
		n := int(binary.BigEndian.Uint32(pngBytes[pos : pos+4]))
		if n < 0 {
			return nil, fmt.Errorf("invalid chunk length")
		}
		typ := string(pngBytes[pos+4 : pos+8])
		chunkStart := pos + 8
		chunkEnd := chunkStart + n
		if chunkEnd+4 > len(pngBytes) {
			return nil, fmt.Errorf("truncated chunk data")
		}
		data := append([]byte(nil), pngBytes[chunkStart:chunkEnd]...)
		out = append(out, pngChunk{Type: typ, Data: data})
		pos = chunkEnd + 4 // skip crc
		if typ == "IEND" {
			break
		}
	}
	return out, nil
}

func firstChunkData(chunks []pngChunk, typ string) []byte {
	for _, ch := range chunks {
		if ch.Type == typ {
			return ch.Data
		}
	}
	return nil
}

func allChunkData(chunks []pngChunk, typ string) [][]byte {
	out := make([][]byte, 0, 4)
	for _, ch := range chunks {
		if ch.Type == typ {
			out = append(out, ch.Data)
		}
	}
	return out
}

func writePNGChunk(buf *bytes.Buffer, typ string, data []byte) {
	if len(typ) != 4 {
		panic("png chunk type must be 4 bytes")
	}
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(data)))
	buf.Write(lenBuf[:])
	buf.WriteString(typ)
	if len(data) > 0 {
		buf.Write(data)
	}
	crc := crc32.NewIEEE()
	crc.Write([]byte(typ))
	if len(data) > 0 {
		crc.Write(data)
	}
	binary.BigEndian.PutUint32(lenBuf[:], crc.Sum32())
	buf.Write(lenBuf[:])
}
