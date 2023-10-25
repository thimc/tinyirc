package main

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
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
)

var (
	host    = flag.String("h", "irc.libera.chat", "host")
	port    = flag.Int("p", 6667, "port")
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

	// TODO: Get rid of magic numbers
	if *usetls && port == 6667 {
		port = 6697
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
	} else {
		if pass != "" {
			if err := sendCommand(dial, fmt.Sprintf("PASS %s", pass)); err != nil {
				return nil, err
			}
		}
	}

	if err := sendCommand(dial, fmt.Sprintf("NICK %s", nick)); err != nil {
		return nil, err
	}
	if err := sendCommand(dial, fmt.Sprintf("USER %s localhost %s :%s", nick, host, nick)); err != nil {
		return nil, err
	}

	return dial, nil
}

func sendCommand(conn net.Conn, message string) error {
	msgFormatted := fmt.Sprintf("%s\r\n", message)
	bytesWrote, err := conn.Write([]byte(msgFormatted))
	if err != nil {
		return err
	}
	if bytesWrote != len(msgFormatted) {
		return fmt.Errorf("Unexpected error, could not write the whole message")
	}

	return nil
}

func parseIRCMessage(conn net.Conn, message string) {
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
		sendCommand(conn, fmt.Sprintf("PONG :%s", trailing))
	case "PRIVMSG":
		printMessage(params, "<%s> %s", prefix, trailing)
	case "CAP":
		for _, p := range strings.Split(message, " ") {
			if p == "sasl" {
				sendCommand(conn, fmt.Sprintf("CAP REQ :%s", p))
			} else if p == "ACK" || p == "NAK" {
				sendCommand(conn, fmt.Sprintf("AUTHENTICATE %s", saslMech))
			}
		}
	case "AUTHENTICATE":
		printMessage(prefix, ">< %s (%s): %s", command, *nick, saslMech)
		str := []byte(fmt.Sprintf("%s\x00%s\x00%s", *nick, *nick, *pass))
		sendCommand(conn, fmt.Sprintf("AUTHENTICATE %s", base64.StdEncoding.EncodeToString(str)))
	case "005": /* IS SUPPORT */
		/* TODO: The issue we have this is because parseIRCMessage splits on " :" */
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
		sendCommand(conn, "CAP END")
	case "904": /* SASL FAIL */
		printMessage(prefix, ">< %s (%s): %s", command, *nick, "SASL: failed")
		fallthrough
	case "905": /* SASL FAIL - TOO LONG (message exceeds 400 bytes) */
		fallthrough
	case "906": /* SASL ABORTED (client side) */
		sendCommand(conn, "CAP END")
		sendCommand(conn, "QUIT")
		os.Exit(1)
		//	case "907": /* TODO: SASL ALREADY (already authenticated) */
		//	case "907": /* TODO: SASL MECHS (request mechanism for SASL authentication) */
	default:
		printMessage(prefix, ">< %s (%s): %s", command, params, trailing)
		//		log.Printf("\033[1;31m%s\033[0m\n", message)
	}
}

func makeInputReader(inputch chan<- string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		inputch <- strings.TrimSpace(input)
	}
}

func makeOutputReader(conn net.Conn, outputch chan<- string) {
	reader := bufio.NewReader(conn)
	for {
		output, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		outputch <- strings.TrimSpace(output)
	}
}

func privateMessage(conn net.Conn, channel, message string) {
	if channel == "" {
		printMessage("Error", "No channel to send to")
		return
	}
	printMessage(channel, "<%s> %s", *nick, message)
	sendCommand(conn, fmt.Sprintf("PRIVMSG %s :%s", channel, message))
}

func printMessage(channel string, format string, a ...any) {
	fmt.Printf("%-19s: %s %s\n", channel, time.Now().Format(timeFormat), fmt.Sprintf(format, a...))
}

func main() {
	flag.Parse()

	if *nick == "" {
		fmt.Printf("Error: nickname cannot be empty\n")
		os.Exit(1)
	}
	if len(*prompt) > 1 {
		fmt.Printf("Error: the command prefix should only be one character.\n")
		os.Exit(1)
	}

	conn, err := newConnection(*nick, *pass, *host, *port)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	inputch := make(chan string)
	outputch := make(chan string)

	go makeInputReader(inputch)
	go makeOutputReader(conn, outputch)

	for {
		select {
		case input := <-inputch:
			if len(input) < 2 {
				continue
			}
			if input[0] != []byte(*prompt)[0] {
				privateMessage(conn, channelName, input)
				continue
			}

			switch input[1] {
			case 'j':
				channelName = strings.Fields(input)[1]
				sendCommand(conn, fmt.Sprintf("JOIN %s", channelName))
			case 'l':
				if channelName == "" {
					continue
				}
				// TODO: get message from user and send it before parting
				sendCommand(conn, fmt.Sprintf("PART %s :%s", channelName, partingMessage))
			case 'm':
				line := strings.Fields(input[2:])
				privateMessage(conn, line[1], strings.Join(line[2:], " "))
			case 's':
				// TODO: Set default channel/user
			case 'q':
				sendCommand(conn, "QUIT")
			default:
				sendCommand(conn, input[1:])
			}
		case output := <-outputch:
			parseIRCMessage(conn, output)
		}
	}
}
