package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"

	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/encoding"
	"github.com/taybart/log"
)

type regularExpresion string
type state struct {
	index int
	rx    regularExpresion
	file  []string
}

func (rx *regularExpresion) add(rn rune, i int) {
	r := []rune(*rx)
	r = append(r, rune(0))
	copy(r[i+1:], r[i:])
	r[i] = rn
	*rx = regularExpresion(r)
}

func (rx *regularExpresion) remove(index int) {
	r := []rune(*rx)
	i := index
	if i == -1 {
		i = len(r) - 1
	}
	if len(r) > 0 {
		r = append(r[:i-1], r[i:]...)
	}
	*rx = regularExpresion(r)
}

func (rx *regularExpresion) compile() (*regexp.Regexp, error) {
	return regexp.Compile(string(*rx))
}

var filename string

var scr tcell.Screen

func init() {
	flag.StringVar(&filename, "f", "", "file to run on")
}

func initScreen() {
	var err error
	scr, err = tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	encoding.Register()
	if err = scr.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	scr.SetStyle(tcell.StyleDefault.
		Foreground(tcell.ColorDefault).
		Background(tcell.ColorDefault))
	scr.EnableMouse()
	scr.Clear()
}

func main() {
	flag.Parse()

	s := state{
		index: 0,
		rx:    "",
		file:  []string{},
	}

	if filename == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter test text: ")
		text, _ := reader.ReadString('\n')
		s.file = append(s.file, text)
	} else {
		file, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			s.file = append(s.file, line)
		}
	}

	initScreen()

	quit := make(chan bool)
	valid := regexp.MustCompile("[^[:cntrl:]]")
	go func() {
		for {
			draw(s)
			event := scr.PollEvent()
			switch ev := event.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyLeft:
					s.index--
					if s.index < 0 {
						s.index = 0
					}
				case tcell.KeyRight:
					s.index++
					if s.index > len(s.rx) {
						s.index = len(s.rx)
					}
				case tcell.KeyEsc:
					quit <- true
					break
				case tcell.KeyBackspace, tcell.KeyBackspace2:
					s.rx.remove(s.index)
					s.index--
					if s.index < 0 {
						s.index = 0
					}
					continue
				case tcell.KeyCtrlC:
					quit <- true
					break
				}
				switch r := ev.Rune(); r {
				case 'q':
					quit <- true
					break
				default:
					if valid.MatchString(string(r)) {
						s.rx.add(r, s.index)
						s.index++
						continue
					}
				}
			}
		}
	}()

	<-quit
	scr.Fini()
}
func draw(s state) {
	w, _ := scr.Size()
	scr.Clear()
	puts(0, 0, w, string(s.rx), true, tcell.StyleDefault.Foreground(tcell.Color142))
	c := " "
	if s.index < len(s.rx) {
		c = string(s.rx[s.index])
	}
	puts(s.index, 0, w, c, true, tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite))
	reg, err := s.rx.compile()
	if err == nil {
		i := 2
		for _, line := range s.file {
			if reg.MatchString(line) {
				m := reg.FindStringSubmatch(line)
				puts(0, i, w, line, true, tcell.StyleDefault.Foreground(tcell.Color109))

				if len(m) > 1 {
					col := len(line) + 1
					str := " -> "
					puts(col, i, w, str, true, tcell.StyleDefault)
					col += len(str) + 1

					str = "groups {{"
					puts(col, i, w, str, true, tcell.StyleDefault.Foreground(tcell.Color66))
					col += len(str) + 1

					for _, c := range m[1:] {
						puts(col, i, w, c, true, tcell.StyleDefault.Foreground(tcell.Color142))
						col += len(c)
						puts(col, i, w, ", ", true, tcell.StyleDefault.Foreground(tcell.Color142))
						col += 2
					}
					col -= 2

					puts(col, i, w, "}}", true, tcell.StyleDefault.Foreground(tcell.Color66))
				}
				i++
			}
		}
	} else {
		puts(0, 1, w, err.Error(), true, tcell.StyleDefault)
	}
	scr.Show()
}
