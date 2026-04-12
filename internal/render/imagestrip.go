package render

// ExtractFrame returns one frame from a strip image.
// If orientation cannot be determined, it falls back to horizontal for wide
// strips and vertical for tall strips.
func ExtractFrame(strip *PixBuf, numFrames, frame, totalFrames int) *PixBuf {
	return ExtractFrameAligned(strip, numFrames, frame, totalFrames, -1)
}

// ExtractFrameAligned extracts one frame from a strip.
// align: 0=vertical, 1=horizontal, 2=individual-files/no-strip, -1=auto-detect.
func ExtractFrameAligned(strip *PixBuf, numFrames, frame, totalFrames, align int) *PixBuf {
	if strip == nil || strip.Width <= 0 || strip.Height <= 0 {
		return nil
	}

	if numFrames <= 1 || align == 2 {
		return strip.Clone()
	}

	if align < 0 {
		if strip.Width >= strip.Height*numFrames {
			align = 1
		} else {
			align = 0
		}
	}

	idx := frameIndex(frame, totalFrames, numFrames)
	if align == 1 {
		fw := strip.Width / numFrames
		if fw <= 0 {
			return strip.Clone()
		}

		out := NewPixBuf(fw, strip.Height)
		for y := range out.Height {
			for x := range out.Width {
				out.Set(x, y, strip.At(idx*fw+x, y))
			}
		}

		return out
	}

	fh := strip.Height / numFrames
	if fh <= 0 {
		return strip.Clone()
	}

	out := NewPixBuf(strip.Width, fh)
	for y := range out.Height {
		for x := range out.Width {
			out.Set(x, y, strip.At(x, idx*fh+y))
		}
	}

	return out
}

func frameIndex(frame, totalFrames, numFrames int) int {
	if numFrames <= 1 {
		return 0
	}

	if totalFrames <= 1 {
		return 0
	}

	if frame < 0 {
		frame = 0
	}

	if frame >= totalFrames {
		frame = totalFrames - 1
	}

	idx := max(numFrames*frame/(totalFrames-1), 0)

	if idx >= numFrames {
		idx = numFrames - 1
	}

	return idx
}
