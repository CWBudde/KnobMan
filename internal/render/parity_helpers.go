package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"knobman/internal/fileio"
	"knobman/internal/model"
)

// LoadParityDocument loads a sample knob and resolves its textures for parity rendering.
func LoadParityDocument(samplePath, repoRoot string) (*model.Document, []*Texture, error) {
	data, err := os.ReadFile(samplePath)
	if err != nil {
		return nil, nil, err
	}

	doc, err := fileio.Load(data)
	if err != nil {
		return nil, nil, err
	}

	textures, err := ResolveTexturesForParity(doc, repoRoot)
	if err != nil {
		return nil, nil, err
	}

	return doc, textures, nil
}

// ResolveTexturesForParity resolves embedded and file-backed textures for parity tests/generation.
func ResolveTexturesForParity(doc *model.Document, repoRoot string) ([]*Texture, error) {
	if doc == nil {
		return nil, nil
	}

	byName := make(map[string]int)
	textures := make([]*Texture, 0)
	assets := []string{
		filepath.Join(repoRoot, "assets", "textures"),
		filepath.Join(repoRoot, "legacy", "res", "Texture"),
	}
	exts := []string{"", ".png", ".bmp", ".jpg", ".jpeg", ".gif"}

	for i := range doc.Layers {
		ly := &doc.Layers[i]
		name := strings.TrimSpace(ly.Prim.TextureName)
		if name == "" || ly.Prim.TextureDepth.Val == 0 {
			ly.Prim.TextureFile.Val = 0
			continue
		}

		if existing, ok := byName[name]; ok {
			ly.Prim.TextureFile.Val = existing
			continue
		}

		var data []byte
		if len(ly.Prim.EmbeddedTexture) > 0 {
			data = ly.Prim.EmbeddedTexture
		} else {
			for _, base := range assets {
				for _, ext := range exts {
					p := filepath.Join(base, name+ext)
					if ext == "" {
						p = filepath.Join(base, name)
					}
					if file, err := os.ReadFile(p); err == nil {
						data = file
						break
					}
				}
				if len(data) > 0 {
					break
				}
			}
		}

		if len(data) == 0 {
			continue
		}

		tex, err := DecodeTexture(data)
		if err != nil {
			return nil, fmt.Errorf("texture %q: %w", name, err)
		}

		textures = append(textures, tex)
		idx := len(textures)
		byName[name] = idx
		ly.Prim.TextureFile.Val = idx
	}

	return textures, nil
}

// ReadPNGAsRGBA loads PNG file bytes into an RGBA image for comparison.
func ReadPNGAsRGBA(path string) (*image.RGBA, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	return imageToRGBA(img), nil
}

// WritePixBufPNG writes a pixbuf to PNG file path.
func WritePixBufPNG(path string, buf *PixBuf) error {
	if buf == nil || buf.Width <= 0 || buf.Height <= 0 || len(buf.Data) == 0 {
		return fmt.Errorf("invalid pixbuf")
	}
	img := image.NewNRGBA(image.Rect(0, 0, buf.Width, buf.Height))
	if len(img.Pix) < len(buf.Data) {
		return fmt.Errorf("pixbuf too large for image")
	}
	copy(img.Pix, buf.Data)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

func imageToRGBA(img image.Image) *image.RGBA {
	if img == nil {
		return nil
	}
	b := img.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))

	switch src := img.(type) {
	case *image.NRGBA:
		for y := 0; y < b.Dy(); y++ {
			srcY := b.Min.Y + y
			srcOffset := srcY*src.Stride + b.Min.X*4
			dstOffset := y * out.Stride
			copy(out.Pix[dstOffset:dstOffset+b.Dx()*4], src.Pix[srcOffset:srcOffset+b.Dx()*4])
		}
		return out
	case *image.Gray:
		for y := 0; y < b.Dy(); y++ {
			srcY := b.Min.Y + y
			srcOffset := srcY*src.Stride + b.Min.X
			dstOffset := y * out.Stride
			for x := 0; x < b.Dx(); x++ {
				v := src.Pix[srcOffset+x]
				dst := dstOffset + x*4
				out.Pix[dst+0] = v
				out.Pix[dst+1] = v
				out.Pix[dst+2] = v
				out.Pix[dst+3] = 255
			}
		}
		return out
	case *image.RGBA:
		for y := 0; y < b.Dy(); y++ {
			srcY := b.Min.Y + y
			for x := 0; x < b.Dx(); x++ {
				srcX := b.Min.X + x
				srcOffset := srcY*src.Stride + srcX*4
				r := uint32(src.Pix[srcOffset+0])
				g := uint32(src.Pix[srcOffset+1])
				bb := uint32(src.Pix[srcOffset+2])
				a := uint32(src.Pix[srcOffset+3])
				dstOffset := y*out.Stride + x*4

				if a == 0 {
					out.Pix[dstOffset+0] = 0
					out.Pix[dstOffset+1] = 0
					out.Pix[dstOffset+2] = 0
					out.Pix[dstOffset+3] = 0
					continue
				}
				out.Pix[dstOffset+0] = uint8((r*255 + a/2) / a)
				out.Pix[dstOffset+1] = uint8((g*255 + a/2) / a)
				out.Pix[dstOffset+2] = uint8((bb*255 + a/2) / a)
				out.Pix[dstOffset+3] = uint8(a)
			}
		}
		return out
	}

	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			n := color.NRGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			dstOffset := y*out.Stride + x*4
			out.Pix[dstOffset+0] = n.R
			out.Pix[dstOffset+1] = n.G
			out.Pix[dstOffset+2] = n.B
			out.Pix[dstOffset+3] = n.A
		}
	}
	return out
}
