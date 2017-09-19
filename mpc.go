package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type Track struct {
	Title  string
	Artist string
	Album  string
	Uri    string
}

type Mpc struct {
	Host string
	Port int
}

func NewMpc() Mpc {
	return Mpc{Port: 6600, Host: "localhost"}
}

func (m Mpc) baseCommand(command ...string) *exec.Cmd {
	args := []string{"-h", m.Host, "-p", fmt.Sprint(m.Port)}
	for _, a := range command {
		args = append(args, a)
	}
	c := exec.Command("mpc", args...)
	return c
}

func (m Mpc) outputCommand(command ...string) (string, error) {
	o, e := m.baseCommand(command...).CombinedOutput()
	if e != nil {
		log.Print(string(o))
		return "", e
	}
	return string(o), nil
}

func (m Mpc) Current() (Track, error) {
	o, e := m.outputCommand("-f", "%artist%\n%album%\n%title%\n%file%", "current")
	if e != nil {
		return Track{}, e
	}
	l := strings.Split(strings.TrimSpace(o), "\n")
	return Track{Artist: l[0], Album: l[1], Title: l[2], Uri: l[3]}, nil
}

func (m Mpc) Next() error {
	return m.baseCommand("next").Run()
}

func (m Mpc) Previous() error {
	return m.baseCommand("previous").Run()
}

func (m Mpc) Playlist() ([]Track, error) {
	o, e := m.outputCommand("-f", "%artist%\n%album%\n%title%\n%file%", "playlist")
	if e != nil {
		return nil, e
	}
	l := strings.Split(strings.TrimSpace(o), "\n")
	c := len(l) / 4
	i := 0
	p := make([]Track, c)
	for i < c {
		p[i].Artist = l[i*4]
		p[i].Album = l[i*4+1]
		p[i].Title = l[i*4+2]
		p[i].Uri = l[i*4+3]
		i++
	}
	return p, nil
}
