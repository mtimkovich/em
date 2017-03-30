package main

import (
    "bufio"
    "container/list"
    "fmt"
    "log"
    "os"
    "strings"
    "strconv"
    "unicode"
)

type Editor struct {
    buffer *list.List
    filename string
    currentLine int
    modified bool
    err string
}

func NewEditor() *Editor {
    e := new(Editor)
    e.buffer = list.New()

    return e
}

func (e *Editor) Index(idx int) *list.Element {
    i := 0
    for l := e.buffer.Front(); l != nil; l = l.Next() {
        if i == idx {
            return l
        }

        i++
    }

    return nil
}

func (e *Editor) Open(filename string) {
    file, err := os.Open(filename)

    if err != nil {
        log.Fatal(err)
    }

    e.filename = filename
    size := 0

    scanner := bufio.NewScanner(file)

    i := 1
    for scanner.Scan() {
        text := scanner.Text()
        size += len(text) + 1
        e.buffer.PushBack(text)
        i++
    }

    e.currentLine = i - 1

    file.Close()

    fmt.Println(size)
}

func (e *Editor) Print(start, end int, numbers bool) {
    i := 1

    for l := e.buffer.Front(); l != nil; l = l.Next() {
        if i >= start && i <= end {
            if numbers {
                fmt.Printf("%d\t%s\n", i, l.Value)
            } else {
                fmt.Println(l.Value)
            }

            e.currentLine = i
        }

        i++
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
}

func (e *Editor) Insert(line int) {
    input := readLines()
    e.InsertBefore(input, line)

    e.modified = true
}

func (e *Editor) Delete(start, end int) {
    curr := e.Index(start-1)

    for i := start; i <= end; i++ {
        next := curr.Next()
        e.buffer.Remove(curr)
        curr = next
    }

    e.modified = true
}

func (e *Editor) Error(msg string) {
    e.err = msg
    fmt.Println("?")
}

func (e *Editor) Prompt() {
    text := readLine()

    if len(text) == 0 {
        e.Error("invalid address")
        return
    }

    command := rune(text[len(text)-1])
    nrange := text

    if unicode.IsLetter(command) {
        nrange = text[:len(text)-1]
    } else {
        command = 'p'
    }

    nums := strings.Split(nrange, ",")
    start := 0
    end := 0

    if len(nums) == 2 {
        start, _ = strconv.Atoi(nums[0])
        end, _ = strconv.Atoi(nums[1])
    } else if len(nums) == 1 {
        start, _ = strconv.Atoi(nums[0])
        end = start
    }

    if start == 0 || end == 0 {
        start = e.currentLine
        end = e.currentLine
    }

    if start > end || start < 0 || end > e.buffer.Len() {
        e.Error("invalid address")
        return
    }

    switch command {
    case 'p':
        e.Print(start, end, false)
    case 'n':
        e.Print(start, end, true)
    case 'i':
        e.Insert(end)
    case 'd':
        e.Delete(start, end)
    case 'c':
        e.Delete(start, end)
        e.Insert(start)
    case 'h':
        if len(e.err) > 0 {
            fmt.Println(e.err)
        }
    case 'q':
        if e.modified {
            e.Error("warning: file modified")
            e.modified = false
        } else {
            os.Exit(0)
        }
    case 'Q':
        os.Exit(0)
    default:
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
