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

type Buffer struct {
    lines *list.List
    filename string
    currentLine int
    modified bool
    err string
}

func NewBuffer() *Buffer {
    b := new(Buffer)
    b.lines = list.New()

    return b
}

func (b *Buffer) Index(idx int) *list.Element {
    i := 0
    for e := b.lines.Front(); e != nil; e = e.Next() {
        if i == idx {
            return e
        }

        i++
    }

    return nil
}

func (b *Buffer) Open(filename string) {
    file, err := os.Open(filename)

    if err != nil {
        log.Fatal(err)
    }

    b.filename = filename
    size := 0

    scanner := bufio.NewScanner(file)

    i := 1
    for scanner.Scan() {
        text := scanner.Text()
        size += len(text)
        b.lines.PushBack(text)
        i++
    }

    b.currentLine = i - 1

    file.Close()

    fmt.Println(size)
}

func (b *Buffer) Print(start, end int, numbers bool) {
    i := 1

    for e := b.lines.Front(); e != nil; e = e.Next() {
        if i >= start && i <= end {
            if numbers {
                fmt.Printf("%d\t%s\n", i, e.Value)
            } else {
                fmt.Println(e.Value)
            }

            b.currentLine = i
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

func (b *Buffer) InsertBefore(other *list.List, line int) {
    node := b.Index(line-1)

    for i, e := other.Len(), other.Back(); i > 0; i, e = i-1, e.Prev() {
        b.lines.InsertBefore(e.Value, node)
        node = node.Prev()
    }
}

func (b *Buffer) Insert(line int) {
    input := readLines()
    b.InsertBefore(input, line)

    b.modified = true
}

func (b *Buffer) Delete(start, end int) {
    curr := b.Index(start-1)

    for i := start; i <= end; i++ {
        next := curr.Next()
        b.lines.Remove(curr)
        curr = next
    }

    b.modified = true
}

func (b *Buffer) Error(msg string) {
    b.err = msg
    fmt.Println("?")
}

func (b *Buffer) Prompt() {
    text := readLine()

    if len(text) == 0 {
        b.Error("invalid address")
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
        start = b.currentLine
        end = b.currentLine
    }

    if start > end || start < 0 || end > b.lines.Len() {
        b.Error("invalid address")
        return
    }

    switch command {
    case 'p':
        b.Print(start, end, false)
    case 'n':
        b.Print(start, end, true)
    case 'i':
        b.Insert(end)
    case 'd':
        b.Delete(start, end)
    case 'c':
        b.Delete(start, end)
        b.Insert(start)
    case 'h':
        if len(b.err) > 0 {
            fmt.Println(b.err)
        }
    case 'q':
        if b.modified {
            b.Error("warning: file modified")
            b.modified = false
        } else {
            os.Exit(0)
        }
    case 'Q':
        os.Exit(0)
    default:
        b.Error("unknown command")
    }
}

func main() {
    buffer := NewBuffer()

    filename := "README.md"
    buffer.Open(filename)

    for {
        buffer.Prompt()
    }
}
