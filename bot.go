package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"time"
)

const BOT_NAME = "WhatItSaysOnTheBox"
const CHANNEL_NAME = "#WhatItSaysOnTheBox"
const SERVER_ADDRESS = "irc.pirc.pl:6697"

type IrcMessage struct {
	fmt.Stringer
	Prefix  string
	Command string
	Params  []string
}

func (m IrcMessage) String() string {
	s := ""
	if len(m.Prefix) > 0 {
		if m.Prefix[0] != ':' {
			s += ":"
		}
		s += m.Prefix
		s += " "
	}
	s += m.Command
	for _, p := range m.Params {
		s += " "
		s += p //TODO: consider validating
	}
	return s
}

func SplitIrcParams(params string) []string {
	p := []string{}
	for len(params) > 0 {
		params = strings.TrimLeft(params, " ")
		if s := strings.Index(params, " "); params[0] == ':' || s == -1 {
			p = append(p, params)
			params = ""
		} else {
			p = append(p, params[:s])
			params = strings.TrimLeft(params[s:], " ")
		}
	}
	return p
}

func ParseIrcMessage(message string) IrcMessage {
	m := strings.Trim(message, "\r\n")
	msg := IrcMessage{}
	if strings.HasPrefix(m, ":") {
		s := strings.Index(m, " ")
		msg.Prefix = m[:s]
		m = strings.TrimLeft(m[s:], " ")
	}
	s := strings.Index(m, " ")
	msg.Command = m[:s]
	msg.Params = SplitIrcParams(strings.TrimLeft(m[s:], " "))
	return msg
}

func bot(inp <-chan IrcMessage, outp chan<- IrcMessage) {
	registered := false
	inside := false
	m := NewMpc()
	for {
		msg := <-inp
		if !registered {
			outp <- IrcMessage{Command: "USER", Params: []string{BOT_NAME, BOT_NAME, "s:", BOT_NAME}}
			outp <- IrcMessage{Command: "NICK", Params: []string{BOT_NAME}}
			registered = true
		}
		switch msg.Command {
		case "372":
			if !inside {
				outp <- IrcMessage{Command: "JOIN", Params: []string{CHANNEL_NAME}}
				inside = true
			}
		case "NOTICE", "PONG", "001", "002", "003", "004", "005", "396", "251", "252", "254", "255", "265", "266", "332", "333", "353", "366", "375", "376", "MODE", "JOIN":
		case "PING":
			outp <- IrcMessage{Command: "PONG", Params: msg.Params}
		case "PRIVMSG":
			var recipient string
			if msg.Params[0] == CHANNEL_NAME {
				recipient = CHANNEL_NAME
			} else {
				recipient = msg.Prefix[1:strings.Index(msg.Prefix, "!")]
			}
			switch msg.Params[1] {
			case ":mpc next":
				e := m.Next()
				if e != nil {
					log.Print(e)
				}
				time.Sleep(200 * time.Millisecond)
				msg.Params[1] = ":mpc current"
				fallthrough
			case ":mpc previous":
				e := m.Previous()
				if e != nil {
					log.Print(e)
				}
				time.Sleep(200 * time.Millisecond)
				msg.Params[1] = ":mpc current"
				fallthrough
			case ":mpc current":
				c, e := m.Current()
				if e != nil {
					log.Print(e)
				} else {
					outp <- IrcMessage{Command: "PRIVMSG", Params: []string{recipient, fmt.Sprintf(":%v – %v", c.Title, c.Artist)}}
				}
			}
		default:
			log.Printf("unknown command: %#v", msg)
		}
	}
}

func pusher(conn *tls.Conn, outp <-chan IrcMessage) {
	for {
		outmsg := <-outp
		if outmsg.Command != ":::" {
			log.Printf("OUT %#v", outmsg)
			_, err := fmt.Fprint(conn, outmsg.String()+"\r\n")
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func main() {
	conn, err := tls.Dial("tcp", SERVER_ADDRESS, nil)
	if err != nil {
		log.Fatal(err)
		return
	}
	rd := bufio.NewReader(conn)
	defer conn.Close()
	incoming := make(chan IrcMessage) // TODO: consider buffering
	outcoming := make(chan IrcMessage)
	go bot(incoming, outcoming)
	go pusher(conn, outcoming)
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			log.Fatal(err)
			return
		}
		newmsg := ParseIrcMessage(line)
		incoming <- newmsg
	}
}
