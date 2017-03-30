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

    scanner := bufio.NewScanner(file)

    i := 1
    for scanner.Scan() {
        b.lines.PushBack(scanner.Text())
        i++
    }

    b.currentLine = i - 1

    file.Close()
}

func (b *Buffer) Print(start int, end int, numbers bool) {
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

func readLines(multiline bool) string {
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        text := scanner.Text()

        if !multiline {
            break
        }
    }

    return text
}

func (b *Buffer) Insert(line int) {
    node := b.Index(line-1)

    text := readLines(false)

    b.lines.InsertBefore(text, node)
    b.modified = true
}

func (b *Buffer) Error(msg string) {
    b.err = msg
    fmt.Println("?")
}

func (b *Buffer) Prompt() {
    text := readLines(false)

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

    filename := "justin.txt"
    buffer.Open(filename)

    for {
        buffer.Prompt()
    }
}
