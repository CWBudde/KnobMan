package render

import (
	"bytes"
	"errors"
	"fmt"
	"image"
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

	restoreFixtureTransparency(samplePath, doc)

	textures, err := ResolveTexturesForParity(doc, repoRoot)
	if err != nil {
		return nil, nil, err
	}

	return doc, textures, nil
}

func restoreFixtureTransparency(samplePath string, doc *model.Document) {
	if doc == nil {
		return
	}

	clean := filepath.ToSlash(samplePath)
	fixtureDirs := []string{
		"tests/parity/primitives/inputs/",
		"tests/parity/animated/inputs/",
	}

	for _, dir := range fixtureDirs {
		if strings.Contains(clean, "/"+dir) || strings.HasPrefix(clean, dir) {
			doc.Prefs.BkColor.Val.A = 0
			return
		}
	}
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
		if ly.Prim.TextureDepth.Val == 0 {
			ly.Prim.TextureFile.Val = 0
			continue
		}

		if name == "" && len(ly.Prim.EmbeddedTexture) == 0 {
			ly.Prim.TextureFile.Val = 0
			continue
		}

		if name != "" {
			if existing, ok := byName[name]; ok {
				ly.Prim.TextureFile.Val = existing
				continue
			}
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

					file, err := os.ReadFile(p)
					if err == nil {
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
			ly.Prim.TextureFile.Val = 0
			continue
		}

		label := name
		if label == "" {
			label = fmt.Sprintf("embedded-%d", i)
		}

		tex, err := DecodeTexture(data)
		if err != nil {
			return nil, fmt.Errorf("texture %q: %w", label, err)
		}

		textures = append(textures, tex)

		idx := len(textures)
		if name != "" {
			byName[name] = idx
		}

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

	return ImageToRGBA(img), nil
}

// WritePixBufPNG writes a pixbuf to PNG file path.
func WritePixBufPNG(path string, buf *PixBuf) error {
	if buf == nil || buf.Width <= 0 || buf.Height <= 0 || len(buf.Data) == 0 {
		return errors.New("invalid pixbuf")
	}

	img := PixBufToNRGBA(buf)
	if img == nil {
		return errors.New("invalid pixbuf image conversion")
	}

	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}
