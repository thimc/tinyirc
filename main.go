package main

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

const (
	timeFormat     = "2006-01-02 15:04"
	partingMessage = "No rest for the wicked"
	saslMech       = "PLAIN"
	portDefault    = 6667
	portTLS        = 6697
)

var (
	host    = flag.String("h", "irc.libera.chat", "host")
	port    = flag.Int("p", portDefault, "port")
	nick    = flag.String("n", os.Getenv("USER"), "nickname")
	pass    = flag.String("k", "", "password")
	prompt  = flag.String("P", "/", "command prefix")
	usetls  = flag.Bool("t", false, "use TLS")
	usesasl = flag.Bool("s", false, "use SASL")

	channelName = ""
	prefix      = ""
	timeout     = time.Minute * 1
)

func newConnection(nick, pass, host string, port int) (net.Conn, error) {
	var (
		dial    net.Conn
		err     error
		tlsConf = &tls.Config{InsecureSkipVerify: true}
		dialer  = &net.Dialer{Timeout: timeout}
	)
	if *usetls && port == portDefault {
		port = portTLS
	}
	if *usetls {
		dial, err = tls.DialWithDialer(dialer, "tcp", fmt.Sprintf("%s:%d", host, port), tlsConf)
		if err != nil {
			return nil, err
		}
	} else {
		dial, err = net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
		if err != nil {
			return nil, err
		}
	}
	if *usesasl {
		if err := sendCommand(dial, "CAP LS"); err != nil {
			return nil, err
		}
	} else if pass != "" {
		if err := sendCommand(dial, fmt.Sprintf("PASS %s", pass)); err != nil {
			return nil, err
		}
	}
	if err := sendCommand(dial, fmt.Sprintf("NICK %s", nick)); err != nil {
		return nil, err
	}
	return dial, sendCommand(dial, fmt.Sprintf("USER %s localhost %s :%s", nick, host, nick))
}

func sendCommand(conn net.Conn, message string) error {
	n := fmt.Sprintf("%s\r\n", message)
	nw, err := conn.Write([]byte(n))
	if err != nil {
		return err
	}
	if nw != len(n) {
		return fmt.Errorf("Error: could not write the whole message: %q", n)
	}
	return nil
}

func parseInput(conn net.Conn, input string) error {
	if len(input) < 2 {
		return nil
	}
	if input[0] != []byte(*prompt)[0] {
		return privateMessage(conn, channelName, input)
	}
	switch input[1] {
	case 'j':
		channelName = strings.Fields(input)[1]
		return sendCommand(conn, fmt.Sprintf("JOIN %s", channelName))
	case 'l':
		if channelName == "" {
			return nil
		}
		// TODO: get message from user and send it before parting
		return sendCommand(conn, fmt.Sprintf("PART %s :%s", channelName, partingMessage))
	case 'm':
		line := strings.Fields(input[2:])
		return privateMessage(conn, line[1], strings.Join(line[2:], " "))
	case 's':
		// TODO: Set default channel/user
	case 'q':
		return sendCommand(conn, "QUIT")
	default:
		return sendCommand(conn, input[1:])
	}
	return nil
}

func parseMessage(conn net.Conn, message string) error {
	var (
		parts    = strings.SplitN(message, " :", 2)
		header   = parts[0]
		command  string
		params   string
		trailing string
	)
	if len(parts) == 2 {
		trailing = parts[1]
	}

	if strings.HasPrefix(header, ":") {
		spaceIndex := strings.Index(header, " ")
		prefix = header[1:spaceIndex]
		header = header[spaceIndex+1:]
	}
	fields := strings.Fields(header)
	if len(fields) > 0 {
		command = fields[0]
	}
	if len(fields) > 1 {
		params = fields[1]
	}
	if strings.Contains(prefix, "!") {
		index := strings.Index(prefix, "!")
		prefix = prefix[:index]
	}
	switch command {
	//	case "PONG": /* PING feedback */
	case "PING":
		return sendCommand(conn, fmt.Sprintf("PONG :%s", trailing))
	case "PRIVMSG":
		printMessage(params, "<%s> %s", prefix, trailing)
	case "CAP":
		for _, p := range strings.Split(message, " ") {
			if p == "sasl" {
				return sendCommand(conn, fmt.Sprintf("CAP REQ :%s", p))
			} else if p == "ACK" || p == "NAK" {
				return sendCommand(conn, fmt.Sprintf("AUTHENTICATE %s", saslMech))
			}
		}
	case "AUTHENTICATE":
		printMessage(prefix, ">< %s (%s): %s", command, *nick, saslMech)
		str := []byte(fmt.Sprintf("%s\x00%s\x00%s", *nick, *nick, *pass))
		return sendCommand(conn, fmt.Sprintf("AUTHENTICATE %s", base64.StdEncoding.EncodeToString(str)))
	case "005": /* IS SUPPORT */
		/* TODO: The issue we have this is because parseMessage splits on " :" */
		printMessage(prefix, ">< %s (%s): %s", command, *nick, strings.Join(fields[2:], " "))
	case "252": /* */
		fallthrough
	case "253": /* */
		fallthrough
	case "254": /* */
		printMessage(prefix, ">< %s (%s): %s", command, *nick, fields[2]+" "+trailing)
		//	case "900": /* TODO: LOGGED IN */
		//	case "901": /* TODO: LOGGED OUT */
		//	case "902": /* TODO: NICK LOCKED */
		//	case "900": /* TODO: LOGGED IN */
	case "903": /* SASL SUCCESS */
		return sendCommand(conn, "CAP END")
	case "904": /* SASL FAIL */
		printMessage(prefix, ">< %s (%s): %s", command, *nick, "SASL: failed")
		fallthrough
	case "905": /* SASL FAIL - TOO LONG (message exceeds 400 bytes) */
		fallthrough
	case "906": /* SASL ABORTED (client side) */
		if err := sendCommand(conn, "CAP END"); err != nil {
			return err
		}
		if err := sendCommand(conn, "QUIT"); err != nil {
			return err
		}
		os.Exit(1)
		//	case "907": /* TODO: SASL ALREADY (already authenticated) */
		//	case "907": /* TODO: SASL MECHS (request mechanism for SASL authentication) */
	default:
		printMessage(prefix, ">< %s (%s): %s", command, params, trailing)
	}
	return nil
}

func privateMessage(conn net.Conn, channel, message string) error {
	if channel == "" {
		return fmt.Errorf("Error: No channel to send to")
	}
	printMessage(channel, "<%s> %s", *nick, message)
	return sendCommand(conn, fmt.Sprintf("PRIVMSG %s :%s", channel, message))
}

func makeReader(r io.Reader, ch chan<- string) {
	br := bufio.NewReader(r)
	for {
		ln, err := br.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		ch <- strings.TrimSpace(ln)
	}
}

func printMessage(channel string, format string, a ...any) {
	fmt.Printf("%-19s: %s %s\n", channel, time.Now().Format(timeFormat), fmt.Sprintf(format, a...))
}

func main() {
	flag.Parse()
	if *nick == "" {
		fmt.Fprintf(os.Stderr, "Error: nickname cannot be empty\n")
		os.Exit(1)
	}
	if len(*prompt) != 1 {
		fmt.Fprintf(os.Stderr, "Error: the command prefix should only be one character.\n")
		os.Exit(1)
	}
	conn, err := newConnection(*nick, *pass, *host, *port)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	inputch := make(chan string)
	outputch := make(chan string)
	go makeReader(os.Stdin, inputch)
	go makeReader(conn, outputch)
	for {
		select {
		case input := <-inputch:
			if err := parseInput(conn, input); err != nil {
				log.Printf("%s\n", err)
			}
		case output := <-outputch:
			if err := parseMessage(conn, output); err != nil {
				log.Printf("%s\n", err)
			}
		}
	}
}
