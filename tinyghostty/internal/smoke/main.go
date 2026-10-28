// Command smoke writes VT bytes into libghostty-vt and prints the screen.
// It checks the binding under TinyGo native and wasip1 builds.
package main

import (
	"fmt"
	"os"

	"nimbleterminal.dev/tinybubbles/tinyghostty/vt"
)

func main() {
	term, err := vt.New(20, 3)
	if err != nil {
		fmt.Println("new:", err)
		os.Exit(1)
	}
	defer term.Close()
	term.WriteString("tiny\x1b[1;32mghostty\x1b[0m\r\nline 2 界")
	if _, err := term.Update(); err != nil {
		fmt.Println("update:", err)
		os.Exit(1)
	}
	cells := make([]vt.Cell, 20)
	for {
		y, n, ok := term.NextRow(cells, false)
		if !ok {
			break
		}
		line := make([]rune, 0, n)
		for _, c := range cells[:n] {
			switch {
			case c.Flags&vt.Spacer != 0:
			case c.Rune == 0:
				line = append(line, ' ')
			default:
				line = append(line, c.Rune)
			}
		}
		fmt.Printf("%d|%s|\n", y, string(line))
	}
	c := term.Cursor()
	fmt.Printf("cursor %d,%d bold=%v\n", c.X, c.Y, cellsBold(term))
}

func cellsBold(term *vt.Terminal) bool {
	term.Update()
	cells := make([]vt.Cell, 20)
	term.NextRow(cells, false)
	return cells[4].Flags&vt.Bold != 0
}
