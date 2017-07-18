package main

import (
	"errors"
	"regexp"
	"strconv"
)

// These constants...
const MaxUint = ^uint(0)
const MaxInt = int(MaxUint >> 1)
const MinInt = -MaxInt - 1

// ...are needed for this
const InvalidAddr = MinInt
const SkipAddr = InvalidAddr + 1
const NOLINE = InvalidAddr
const NORUNE = 0xfffd

var ErrNoMatch = errors.New("no match")

// ----------
// LineParser
//

// LineParser is a simple two-index parser.
// Main index advances only on successful rule matches
// Secondary index is used in intermediate matching.
type LineParser struct {
	// Main and secondary indices
	mi int
	si int

	text   []rune
	ctx    *Context
	editor *Editor
}

func NewLineParser(e *Editor, t string) *LineParser {
	return &LineParser{
		mi:     0,
		si:     0,
		text:   []rune(t),
		ctx:    nil, // DO NOT forget to initialize
		editor: e,
	}
}

// Advance is called on rule match success
func (lp *LineParser) Advance() {
	lp.mi = lp.si
}

// Rewind is called on rule match failure
func (lp *LineParser) Rewind() {
	lp.si = lp.mi
}

func (lp *LineParser) LookAhead() (r rune, ok bool) {
	if lp.si >= len(lp.text) {
		return NORUNE, false
	}
	return lp.text[lp.si], true
}

func (lp *LineParser) Consume() {
	if lp.si <= len(lp.text) {
		lp.si++
	}
}

func (lp *LineParser) ConsumeAll() {
	for lp.si < len(lp.text) {
		lp.si++
	}
}

func (lp *LineParser) ConsumeSpace() {
outer:
	for {
		la, ok := lp.LookAhead()
		if !ok {
			break
		}
		switch la {
		case '\t', ' ':
			lp.Consume()
		default:
			break outer
		}
	}
	lp.Advance()
}

func (lp *LineParser) Match(r rune) bool {
	la, ok := lp.LookAhead()
	if !ok {
		return false
	}
	if la == r {
		lp.Consume()
		return true
	}
	return false
}

func (lp *LineParser) Parse() (start, end int, cmd rune, text string) {
	if len(lp.text) == 0 { // just Enter
		addr := lp.editor.CurrentAddr() + 1
		return addr, addr, 'p', "p"
	}
	ctx := NewContext(lp, lp.editor)
	lp.ctx = ctx
	_ = addrRange(ctx)
	if lp.si >= len(lp.text) {
		lp.text = append(lp.text, 'p') // default command
	}
	return ctx.fstAddr, ctx.sndAddr, lp.text[lp.si], string(lp.text[lp.si:])
}

// Instead of global state in C
type Context struct {
	fstAddr int
	sndAddr int
	addrCnt int
	lp *LineParser
	e  *Editor
}

func NewContext(lp *LineParser, e *Editor) *Context {
	return &Context{
		fstAddr: InvalidAddr,
		sndAddr: InvalidAddr,
		addrCnt: 0,
		lp: lp,
		e:  e,
	}
}

func invalidAddr(ctx *Context) {
	ctx.e.Error("invalid address")
}

func skipBlanks(ctx *Context) {
	ctx.lp.ConsumeSpace()
}

func isDigit(r rune) bool {
	switch r {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return true
	}
	return false
}

func nextAddr(ctx *Context) (int, bool) { // addr, ok
	skipBlanks(ctx)

	addr := ctx.e.CurrentAddr()
	first := true

	for {
		r, ok := ctx.lp.LookAhead()
		if !ok {
			r = NORUNE
		}

		if isDigit(r) {
			if !first {
				invalidAddr(ctx)
				return InvalidAddr, false
			}
			addr, ok = readInt(ctx)
			if !ok {
				invalidAddr(ctx)
				return InvalidAddr, false
			}
		} else {
			switch r {
			case '+', '\t', ' ', '-':
				ctx.lp.Consume()
				ctx.lp.ConsumeSpace()
				r2, ok := ctx.lp.LookAhead()
				if !ok {
					r2 = NORUNE
				}
				if isDigit(r2) {
					if n, ok := readInt(ctx); ok {
						if r == '-' {
							n = -n
						}
						addr += n
					} else {
						return addr, false
					}
				} else if r == '+' {
					addr++
				} else if r == '-' {
					addr--
				}
			case '.', '$':
				if !first {
					invalidAddr(ctx)
					return addr, false
				}
				ctx.lp.Consume()
				addr = ctx.e.LastAddr()
				if r == '.' {
					addr = ctx.e.CurrentAddr()
				}
			case '/', '?':
				if  !first {
					invalidAddr(ctx)
					return addr, false
				}
				rx, ok := matchDelimitedRegexp(ctx, r)
				if !ok {
					return addr, false
				}

				if len(rx) > 0 {
					regexp, err := regexp.Compile(rx)
					if err != nil {
						return addr, false
					}
					ctx.e.pattern = regexp
				}

				var err error
				addr, err = ctx.e.Search(r == '/')
				if err != nil {
					return addr, false
				}
			case '%', ',', ';':
				if first {
					ctx.lp.Consume()
					ctx.addrCnt++
					ctx.sndAddr = 1
					if r == ';' {
						ctx.sndAddr = ctx.e.CurrentAddr() // current line
					}
					addr = ctx.e.LastAddr()
					break // from switch, read next addr in for loop
				}
				fallthrough
			default:
				if addr < 0 || addr > ctx.e.LastAddr() {
					invalidAddr(ctx)
					return addr, false
				}
				ctx.addrCnt++
				return addr, true
			} // switch r
		}
		first = false
	} // for
}

// addrRange returns number of addresses read
func addrRange(ctx *Context) int {
	addr := InvalidAddr
	ok := false
	ctx.addrCnt = 0

	ctx.fstAddr = ctx.e.CurrentAddr()
	ctx.sndAddr = ctx.fstAddr

	for {
		addr, ok = nextAddr(ctx)
		if !ok {
			break
		}
		ctx.fstAddr = ctx.sndAddr
		ctx.sndAddr = addr
		r, ok := ctx.lp.LookAhead()
		if !ok || (r != ',' && r != ';') {
			break
		}
		if r == ';' {
			ctx.e.line = addr
		}
		ctx.lp.Consume()
	}

	if ctx.addrCnt == 1 || ctx.sndAddr != addr {
		ctx.fstAddr = ctx.sndAddr
	}
	return ctx.addrCnt
}

func readInt(c *Context) (int, bool) { // n, ok
	num := ""
	cur := c.lp.si
	for {
		r, ok := c.lp.LookAhead()
		if !ok || !isDigit(r){
			break
		}
		c.lp.Consume()
	}
	num = string(c.lp.text)[cur:c.lp.si]
	n, err := strconv.Atoi(num)
	if err != nil {
		return InvalidAddr, false
	}
	c.lp.Advance()
	return n, true
}

func matchDelimitedRegexp(c *Context, r rune) (string, bool) {
	ok := c.lp.Match(r)
	if !ok {
		return "", false
	}

	raw := []rune("")
	escaped := false
	for {
		la, ok := c.lp.LookAhead()
		if !ok {
			break // break loop if end of input
		}
		c.lp.Consume()

		if la == '\\' {
			if escaped {
				raw = append(raw, '\\')
				escaped = false
			} else {
				escaped = true
			}
			continue
		}

		if la == r {
			if !escaped {
				break // break loop if unescaped delimiter met
			}
		}

		escaped = false
		raw = append(raw, la)
	}

	// No point in matching delimiter now
	// Either we have already matched it in loop or reached end of input
	rx := string(raw)
	return rx, true
}
