package render

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ResolveDynamicText matches JKnobMan's DynamicText.Get behavior for PrimText.
func ResolveDynamicText(s string, frame, totalFrames int) string {
	dt := newDynamicText(s)
	return dt.get(frame, totalFrames)
}

// SubstituteFrameCounters is kept for compatibility with older callers.
func SubstituteFrameCounters(s string, frame, totalFrames int) string {
	return ResolveDynamicText(s, frame, totalFrames)
}

type dynamicText struct {
	str   string
	p     int
	pi    int
	total int
	fmt   string
	fx    int
	pn1   float64
	pn2   float64
	pn3   float64
}

func newDynamicText(s string) dynamicText {
	dt := dynamicText{
		str: s,
		fmt: "%d",
	}
	dt.total = dt.count()

	return dt
}

func (dt *dynamicText) get(frame, totalFrames int) string {
	if totalFrames <= 1 {
		return dt.getItem(0)
	}

	return dt.getItem(dt.total * frame / (totalFrames - 1))
}

func (dt *dynamicText) getItem(n int) string {
	p := 0
	dt.pi = 0
	item := ""

	if n >= dt.total {
		n = dt.total - 1
	}

	for n >= 0 {
		item = dt.getNext(p)
		p = dt.p
		n--
	}

	return item
}

func (dt *dynamicText) getNext(pStart int) string {
	var out strings.Builder
	p := pStart

	for {
		if dt.getChar(p) == '(' {
			if i := dt.checkNum(p); i >= 0 {
				val := dt.pn1 + float64(dt.pi)*dt.pn3
				if dt.pn2 < dt.pn1 {
					val = dt.pn1 - float64(dt.pi)*dt.pn3
				}

				out.WriteString(dt.wsprintf(val))
				dt.pi++

				dt.p++
				for dt.p < len(dt.str) && dt.str[dt.p] != ',' {
					out.WriteByte(dt.str[dt.p])
					dt.p++
				}

				if dt.pi > i {
					dt.pi = 0

					dt.p = dt.skip(dt.p)
					if dt.p < len(dt.str) && dt.str[dt.p] == ',' {
						dt.p++
					}
				} else {
					dt.p = pStart
				}

				return out.String()
			}
		}

		if p >= len(dt.str) {
			break
		}

		if dt.str[p] == ',' {
			p++
			break
		}

		out.WriteByte(dt.str[p])
		p++
	}

	dt.p = p

	return out.String()
}

func (dt *dynamicText) count() int {
	total := 1

	for p := 0; p < len(dt.str); p++ {
		switch dt.getChar(p) {
		case ',':
			total++
		case '(':
			if j := dt.checkNum(p); j > 0 {
				p = dt.p
				total += j
			}
		}
	}

	return total
}

func (dt *dynamicText) checkNum(p int) int {
	var iParen int
	dt.fx = 0
	dt.fmt = "%d"

	r1 := dt.getANum(p + 1)
	if dt.getChar(dt.p) != ':' {
		return -1
	}

	r2 := dt.getANum(dt.p + 1)
	r3 := 1.0

	if dt.getChar(dt.p) == ':' {
		c := dt.getChar(dt.p + 1)
		if dt.isDigit(c) || c == '.' {
			r3 = dt.getANum(dt.p + 1)
			if r3 == 0 {
				r3 = 1
			}
		}

		if dt.getChar(dt.p) == ':' {
			dt.p++

			dt.fmt = ""
			for dt.getChar(dt.p) != ')' && dt.getChar(dt.p) != ':' && dt.p < len(dt.str) {
				dt.fmt += string(dt.getChar(dt.p))
				dt.p++
			}
		}

		if dt.getChar(dt.p) == ':' {
			dt.p++
			dt.fx = dt.p
			iParen = 0

			for dt.p < len(dt.str) && dt.getChar(dt.p) != ':' {
				if dt.getChar(dt.p) == '(' {
					iParen++
				}

				if dt.getChar(dt.p) == ')' {
					iParen--
					if iParen < 0 {
						break
					}
				}

				dt.p++
			}
		}
	}

	if dt.getChar(dt.p) != ')' {
		return -1
	}

	dt.pn1 = r1
	dt.pn2 = r2
	dt.pn3 = r3

	return int(math.Abs(r1-r2) / r3)
}

func (dt *dynamicText) wsprintf(val float64) string {
	if dt.fx != 0 {
		val = dt.eval(dt.fx, val)
	}

	var out strings.Builder
	pcs := 4
	width := 0
	zeroPad := false
	showPlus := false

	for i := 0; i < len(dt.fmt); i++ {
		if dt.fmt[i] != '%' {
			out.WriteByte(dt.fmt[i])
			continue
		}

		i++
		if i >= len(dt.fmt) {
			break
		}

		if dt.fmt[i] == '+' {
			showPlus = val >= 0

			i++
			if i >= len(dt.fmt) {
				break
			}
		}

		if dt.fmt[i] == '0' {
			zeroPad = true

			i++
			if i >= len(dt.fmt) {
				break
			}
		}

		if dt.isDigit(dt.fmt[i]) {
			width = int(dt.fmt[i] - '0')

			i++
			if i >= len(dt.fmt) {
				break
			}
		}

		if dt.fmt[i] == '.' {
			i++
			if i >= len(dt.fmt) {
				break
			}

			pcs = 0
			if dt.isDigit(dt.fmt[i]) {
				pcs = int(dt.fmt[i] - '0')

				i++
				if i >= len(dt.fmt) {
					break
				}
			}
		}

		var strv string

		switch dt.fmt[i] {
		case 'd':
			strv = strconv.Itoa(int(val))
		case 'f':
			strv = fmt.Sprintf("%."+strconv.Itoa(pcs)+"f", val)
		case 'x':
			strv = fmt.Sprintf("%x", int(val))
		case 'X':
			strv = fmt.Sprintf("%X", int(val))
		}

		if width != 0 {
			pad := "        "
			if zeroPad {
				pad = "00000000"
			}

			strv = pad + strv
			strv = strv[len(strv)-width:]
		}

		if showPlus {
			strv = "+" + strv
		}

		out.WriteString(strv)
	}

	return out.String()
}

func (dt *dynamicText) eval(p int, val float64) float64 {
	return dt.eval3(p, val)
}

func (dt *dynamicText) eval3(p int, val float64) float64 {
	d := dt.eval2(p, val)
	for {
		switch dt.getChar(dt.p) {
		case '+':
			d += dt.eval2(dt.p+1, val)
		case '-':
			d -= dt.eval2(dt.p+1, val)
		default:
			return d
		}
	}
}

func (dt *dynamicText) eval2(p int, val float64) float64 {
	d := dt.eval1(p, val)
	for {
		switch dt.getChar(dt.p) {
		case '*':
			d *= dt.eval1(dt.p+1, val)
		case '/':
			d /= dt.eval1(dt.p+1, val)
		default:
			return d
		}
	}
}

func (dt *dynamicText) eval1(p int, val float64) float64 {
	if p < 0 || p >= len(dt.str) {
		dt.p = p
		return 0
	}

	switch {
	case strings.HasPrefix(dt.str[p:], "pow("):
		d := dt.eval(p+4, val)
		if dt.getChar(dt.p) == ',' {
			dt.p++
		}

		d = math.Pow(d, dt.eval(dt.p, val))
		if dt.getChar(dt.p) == ')' {
			dt.p++
		}

		return d
	case strings.HasPrefix(dt.str[p:], "exp("):
		d := dt.eval(p+4, val)
		if dt.getChar(dt.p) == ')' {
			dt.p++
		}

		return math.Exp(d)
	case strings.HasPrefix(dt.str[p:], "log("):
		d := dt.eval(p+4, val)
		if dt.getChar(dt.p) == ')' {
			dt.p++
		}

		return math.Log(d)
	case strings.HasPrefix(dt.str[p:], "log10("):
		d := dt.eval(p+6, val)
		if dt.getChar(dt.p) == ')' {
			dt.p++
		}

		return math.Log10(d)
	case strings.HasPrefix(dt.str[p:], "sqrt("):
		d := dt.eval(p+5, val)
		if dt.getChar(dt.p) == ')' {
			dt.p++
		}

		return math.Sqrt(d)
	default:
		return dt.eval0(p, val)
	}
}

func (dt *dynamicText) eval0(p int, val float64) float64 {
	switch dt.getChar(p) {
	case 'x':
		dt.p = p + 1
		return val
	case '(':
		d := dt.eval(p+1, val)
		if dt.getChar(dt.p) == ')' {
			dt.p++
		}

		return d
	case '-':
		return -dt.eval0(p+1, val)
	case '+':
		return dt.eval0(p+1, val)
	default:
		return dt.getANum(p)
	}
}

func (dt *dynamicText) getANum(p int) float64 {
	sign := 1.0
	frac := 0.0
	n1 := 0

	p = dt.skip(p)
	if dt.getChar(p) == '-' {
		sign = -1
		p++
	}

	p = dt.skip(p)
	for dt.isDigit(dt.getChar(p)) {
		n1 = n1*10 + int(dt.getChar(p)-'0')
		p++
	}

	if dt.getChar(p) == '.' {
		dp := 0.1
		for p++; dt.isDigit(dt.getChar(p)); p++ {
			frac += dp * float64(dt.getChar(p)-'0')
			dp *= 0.1
		}
	}

	dt.p = dt.skip(p)
	dt.pn3 = sign * (float64(n1) + frac)

	return dt.pn3
}

func (dt *dynamicText) getChar(p int) byte {
	if p >= len(dt.str) || p < 0 {
		return 0
	}

	return dt.str[p]
}

func (dt *dynamicText) skip(p int) int {
	for dt.getChar(p) == ' ' {
		p++
	}

	return p
}

func (dt *dynamicText) isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}
