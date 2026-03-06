package render

import (
	"math"
	"strconv"
	"strings"
)

// SubstituteFrameCounters replaces "(start:end)" patterns with frame-based
// zero-padded values. Example: "(1:99)".
func SubstituteFrameCounters(s string, frame, totalFrames int) string {
	if totalFrames < 1 {
		totalFrames = 1
	}
	var out strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '(' {
			out.WriteByte(s[i])
			i++
			continue
		}
		end := strings.IndexByte(s[i:], ')')
		if end <= 0 {
			out.WriteByte(s[i])
			i++
			continue
		}
		expr := s[i+1 : i+end]
		colon := strings.IndexByte(expr, ':')
		if colon < 0 {
			out.WriteByte(s[i])
			i++
			continue
		}
		aTxt := strings.TrimSpace(expr[:colon])
		bTxt := strings.TrimSpace(expr[colon+1:])
		a, errA := strconv.Atoi(aTxt)
		b, errB := strconv.Atoi(bTxt)
		if errA != nil || errB != nil {
			out.WriteByte(s[i])
			i++
			continue
		}

		val := a
		if totalFrames > 1 {
			t := float64(frame) / float64(totalFrames-1)
			val = a + int(math.Round(float64(b-a)*t))
		}

		w := max(countDigits(aTxt), countDigits(bTxt))
		if w == 0 {
			w = max(len(strings.TrimPrefix(strconv.Itoa(absInt(a)), "-")), len(strings.TrimPrefix(strconv.Itoa(absInt(b)), "-")))
		}

		neg := val < 0
		if neg {
			val = -val
		}
		num := strconv.Itoa(val)
		for len(num) < w {
			num = "0" + num
		}
		if neg {
			num = "-" + num
		}
		out.WriteString(num)
		i += end + 1
	}
	return out.String()
}

func countDigits(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if s[0] == '+' || s[0] == '-' {
		s = s[1:]
	}
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			n++
		}
	}
	return n
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
