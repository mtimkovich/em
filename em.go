package main

import (
    "bufio"
    "container/list"
    "errors"
    "fmt"
    "os"
    "regexp"
    "strconv"
    "strings"
)

func ExplicitEscapes(s string) (result string) {
	result = s
	replacements := map[string]string{
		"\\": "\\\\",
		"\t": "\\t",
		"\a": "\\a",
		"\b": "\\b",
		"\r": "\\r",
		"\n": "\\n",
	}
	for k, v := range(replacements) {
		result = strings.Replace(result, k, v, -1)
	}
	result += "$"
	return
}

type Editor struct {
    buffer *list.List
    filename string
    line int
    modified bool
    err string
    commands map[rune]func(int, int, rune, string)
	pattern *regexp.Regexp
}

func NewEditor() *Editor {
    e := &Editor{}
    e.buffer = list.New()
    e.line = 0
	e.pattern = nil

    e.commands = map[rune]func(int, int, rune, string){
        'p': e.Print,
        'n': e.Print,
		'l': e.Print,
        'i': e.Insert,
        'a': e.Insert,
        'd': e.Delete,
        'c': e.Change,
        'e': e.OpenWrapper,
        'E': e.OpenWrapper,
        's': e.ReSub,
        'w': e.Write,
        'h': e.Help,
        'q': e.Quit,
        'Q': e.Quit,
    }

    return e
}

func (e *Editor) isModified() bool {
    if e.modified {
        e.Error("warning: file modified")
        e.modified = false
        return true
    } else {
        return false
    }
}

func (e *Editor) Index(idx int) *list.Element {
    for i, l := 0, e.buffer.Front(); l != nil; i, l = i+1, l.Next() {
        if i == idx {
            return l
        }
    }

    return nil
}

func (e *Editor) LastAddr() int {
	return e.buffer.Len()
}

func (e *Editor) CurrentAddr() int {
	return e.line
}

func (e *Editor) IsBufferEmpty() bool {
	return e.buffer.Len() <= 0
}

// Search uses Editor.pattern for matching
func (e *Editor) Search(fwd bool) (num int, err error) {
	// helper
	cyclicNextElem := func (el *list.Element, l *list.List, i int, fwd bool) (e *list.Element, idx int) {
		idx = i
		if fwd {
			e = el.Next()
			idx++
			if e == nil {
				e = l.Front()
				idx = 1
			}
		} else {
			e = el.Prev()
			idx--
			if e == nil {
				e = l.Back()
				idx = l.Len()
			}
		}
		return
	}

	// check previous regexp pattern
	if e.pattern == nil {
		return InvalidAddr, errors.New("no previous pattern")
	}

	rx := e.pattern
	num = e.line

	start := e.Index(num-1)
	if start == nil {
		return InvalidAddr, errors.New("wrong line number")
	}

	// Start searching from this line
	fromLine := start

	// we change fromLine because we do not match the line we start at
	fromLine, num = cyclicNextElem(start, e.buffer, num, fwd)

	current := fromLine
	for {
		value := current.Value.(string)
		// check for match
		if rx.MatchString(value) {
			return num, nil
		}

		// if no match get next line
		current, num = cyclicNextElem(current, e.buffer, num, fwd)

		// we have wrapped around without matching
		if current == start {
			return InvalidAddr, ErrNoMatch
		}
	} // for
}

func (e *Editor) OpenWrapper(start, end int, cmd rune, text string) {
    args := strings.Split(text, " ")
    filename := ""

    if len(args) == 1 {
        return
    }

    if cmd != 'E' {
        if e.isModified() {
            return
        }
    }

    filename = args[1]
    e.Open(filename)
}

func (e *Editor) Open(filename string) {
    file, err := os.Open(filename)
    defer file.Close()

    if err != nil {
        fmt.Println(err)
        e.Error("cannot open input file")
        return
    }

    e.buffer = list.New()
    e.filename = filename
    e.modified = false
    size := 0

    scanner := bufio.NewScanner(file)

    for i := 1; scanner.Scan(); i++ {
        text := scanner.Text()
        size += len(text) + 1
        e.buffer.PushBack(text)

        e.line = i
    }

    fmt.Println(size)
}

func (e *Editor) Write(start, end int, cmd rune, text string) {
    args := strings.Split(text, " ")

	if len(args[0]) > 1 && args[0] != "wq" {
		e.Error("unexpected command suffix")
		return
	}

	filename := e.filename

	// if filename is given
    if len(args) > 1 {
        filename = args[1]
    }

	// if there was no previous filename set it
	if len(e.filename) == 0 && len(filename) > 0 {
		e.filename = filename
	}

	// if neither previous nor new filename exist - error
    if len(filename) == 0 {
        e.Error("no current filename")
        return
    }

    file, err := os.Create(e.filename)
    defer file.Close()

    if err != nil {
        fmt.Println(err)
        e.Error("cannot write to file")
        return
    }

    size := 0

    for l := e.buffer.Front(); l != nil; l = l.Next() {
        text := l.Value.(string)
        count, _ := file.WriteString(text + "\n")
        size += count
    }

    e.modified = false
    fmt.Println(size)

	// wq shortcut
	if args[0] == "wq" {
		e.Quit(start, end, 'q', "")
	}
}

func (e *Editor) Print(start, end int, cmd rune, text string) {
	if e.IsBufferEmpty() {
		e.Error("invalid address")
		return
	}
    for i, l := 1, e.buffer.Front(); l != nil; i, l = i+1, l.Next() {
        if i >= start && i <= end {
            if cmd == 'n' {
                fmt.Printf("%d\t%s\n", i, l.Value)
            } else if cmd == 'l' {
				text := l.Value.(string)
                fmt.Println(ExplicitEscapes(text))
			} else {
				fmt.Println(l.Value)
			}

            e.line = i
        }
    }
}

func readLine() string {
    scanner := bufio.NewScanner(os.Stdin)
    scanner.Scan()
    return scanner.Text()
}

func readLines() *list.List {
    input := list.New()

    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        text := scanner.Text()

        if text == "." {
            break
        }

        input.PushBack(text)
    }

    return input
}

func (e *Editor) InsertBefore(other *list.List, line int) {
    node := e.Index(line-1)

    for i, l := other.Len(), other.Back(); i > 0; i, l = i-1, l.Prev() {
        e.buffer.InsertBefore(l.Value, node)
        node = node.Prev()
    }

    e.setLine(e.line + other.Len() - 1)
}

func (e *Editor) InsertAfter(other *list.List, line int) {
    node := e.Index(line-1)

    for i, l := 0, other.Front(); i < other.Len(); i, l = i+1, l.Next() {
        e.buffer.InsertAfter(l.Value, node)
        node = node.Next()
        e.setLine(e.line+1)
    }

    e.setLine(other.Len() + e.line)
}

func (e *Editor) Insert(start, end int, cmd rune, text string) {
    input := readLines()
    e.setLine(end)

    if e.buffer.Len() == 0 {
        e.buffer.PushBackList(input)
        e.setLine(e.line + input.Len())
    } else {
        if cmd == 'i' {
            // edge case
            if end >= e.buffer.Len() {
                e.buffer.PushBackList(input)
                e.setLine(e.line + input.Len())
            } else {
                e.InsertBefore(input, end)
            }
        } else {
            e.InsertAfter(input, end)
        }
    }

    e.modified = true
}

func (e *Editor) setLine(line int) {
    if line > e.buffer.Len() {
        e.line = e.buffer.Len()
    } else if line <= 0 {
        e.line = 1
    } else {
        e.line = line
    }
}

func (e *Editor) Delete(start, end int, cmd rune, text string) {
	if e.IsBufferEmpty() {
		e.Error("invalid address")
		return
	}
    curr := e.Index(start-1)
	if curr == nil {
		e.Error("invalid address")
		return
	}

    for i := start; i <= end; i++ {
        next := curr.Next()
        e.buffer.Remove(curr)
        curr = next
    }

    e.setLine(start)
    e.modified = true
}

func (e *Editor) Change(start, end int, cmd rune, text string) {
	if e.IsBufferEmpty() {
		e.Error("invalid address")
		return
	}
    e.Delete(start, end, cmd, text)
    e.Insert(start, end, 'i', text)
}

func (e *Editor) Error(msg string) {
    e.err = msg
    fmt.Println("?")
}

func (e *Editor) replaceMacros(text string) string {
    macros := map[string]int{
        ".": e.line,
        "+": e.line+1,
        "-": e.line-1,
        "$": e.buffer.Len(),
    }

    for key, value := range macros {
        text = strings.Replace(text, key, strconv.Itoa(value), -1)
    }

    return text
}

func (e *Editor) ReSub(start, end int, cmd rune, text string) {
	sep := "/"
	if len(text) > 1 {
		sep = string(text[1])
	}
	if sep == " " || sep == "\t" || sep == "\n" || sep == "\r" {
		e.Error("invalid pattern delimiter")
		return
	}
	parts := strings.Split(text, sep)
	lenparts := len(parts)

	if lenparts < 2 || lenparts > 4 { // not taking single 's' into account
		e.Error("invalid command syntax")
		return
	}

    match := parts[1]
	replace := ""
	if lenparts > 2 {
		replace = parts[2]
	}
	flags := ""
	if lenparts > 3 {
		flags = parts[3]
	}

    if strings.ContainsRune(flags, 'i') {
        match = "(?i)" + match
    }

	global := false
	if strings.ContainsRune(flags, 'g') {
		global = true
	}

    re, err := regexp.Compile(match)

    if err != nil {
        e.Error("invalid regexp")
        return
    }

    for i, l := 1, e.buffer.Front(); l != nil; i, l = i+1, l.Next() {
		if i > end {
			break
		}
		if i >= start && i <= end {
			line := l.Value.(string)
			if global {
				l.Value = re.ReplaceAllString(line, replace)
			} else {
				pos := re.FindStringIndex(line)
				if pos != nil {
					head := line[:pos[1]]
					tail := line[pos[1]:]
					l.Value = re.ReplaceAllString(head, replace) + tail
				}
			}

            e.line = i
        }
    }

	if lenparts == 3 || strings.ContainsRune(flags, 'p') { // 'p' flag by default
		e.Print(end, end, 'p', text)
	}

    e.modified = true
}

func (e *Editor) Quit(start, end int, cmd rune, text string) {
    if cmd == 'Q' || !e.isModified() {
        os.Exit(0)
    }
}

func (e *Editor) Help(start, end int, cmd rune, text string) {
    if len(e.err) > 0 {
        fmt.Println(e.err)
    }
}

func (e *Editor) Parse(text string) (int, int, string) {
    if len(text) == 0 {
        return e.line+1, e.line+1, "p"
    }

    index := -1
    for i, c := range text {
        if _, ok := e.commands[c]; ok {
            index = i
            break
        }
    }

    if index == 0 {
        return e.line, e.line, text
    }

    var nrange, rest string

    if index == -1 {
        nrange = text
    } else {
        nrange = text[:index]
    }

    nrange = e.replaceMacros(nrange)

    nums := strings.Split(nrange, ",")
    start := 0
    end := 0

    if nrange == "," || nrange == "%" {
        start = 1
        end = e.buffer.Len()
    } else if len(nums) == 2 {
        start, _ = strconv.Atoi(nums[0])
        end, _ = strconv.Atoi(nums[1])
    } else if len(nums) == 1 {
        start, _ = strconv.Atoi(nums[0])
        end = start
    }

    if start == 0 && end == 0 {
        // Invalid input
    } else if index == -1 {
        rest = "p"
    } else {
        rest = text[index:]
    }

    return start, end, rest
}

func (e *Editor) Prompt() {
    text := readLine()
	p := NewLineParser(e, text)

	start, end, cmd, text := p.Parse()

    if text == "" {
        e.Error("unknown command")
        return
    }

    // Special check when working on an empty buffer
    if (e.buffer.Len() != 0 && start != 1) &&
           (start < 1 || end > e.buffer.Len() ||
           start > end) {
        e.Error("invalid address")
        return
    }

    if fn, ok := e.commands[cmd]; ok {
        fn(start, end, cmd, text)
    } else {
        e.Error("unknown command")
    }
}

func main() {
    editor := NewEditor()

    if len(os.Args) > 1 {
        editor.Open(os.Args[1])
    }

    for {
        editor.Prompt()
    }
}
