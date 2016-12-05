package command

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/wenlian/remote-tail/console"
	"github.com/wenlian/remote-tail/ssh"
)

type Command struct {
	Host   string
	User   string
	Path   string
	Script string
	Stdout io.Reader
	Stderr io.Reader
	Server Server
}

// The message used by channel to transport log line by line
type Message struct {
	Host    string
	Path    string
	Content string
}

// Create a new command
func NewCommand(server Server, logPath string) (cmd *Command) {
	cmd = &Command{
		Host:   server.Hostname,
		User:   server.User,
		Path:   logPath,
		Script: fmt.Sprintf("tail -f %s", logPath),
		Server: server,
	}
	if !strings.Contains(cmd.Host, ":") {
		cmd.Host = cmd.Host + ":" + strconv.Itoa(server.Port)
	}
	return
}

// Execute the remote command
func (cmd *Command) Execute(output chan Message) error {
	var wg sync.WaitGroup

	client := &ssh.Client{
		Host:     cmd.Host,
		User:     cmd.User,
		Password: cmd.Server.Password,
	}

	if err := client.Connect(); err != nil {
		panic(fmt.Sprintf("[%s] unable to connect: %s", cmd.Host, err))
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		panic(fmt.Sprintf("[%s] unable to create session: %s", cmd.Host, err))
		return err
	}
	defer session.Close()

	if err := session.RequestPty("xterm", 80, 40, *ssh.CreateTerminalModes()); err != nil {
		panic(fmt.Sprintf("[%s] unable to create pty: %v", cmd.Host, err))
		return err
	}

	cmd.Stdout, err = session.StdoutPipe()
	if err != nil {
		panic(fmt.Sprintf("[%s] redirect stdout failed: %s", cmd.Host, err))
		return err
	}

	cmd.Stderr, err = session.StderrPipe()
	if err != nil {
		panic(fmt.Sprintf("[%s] redirect stderr failed: %s", cmd.Host, err))
		return err
	}
	wg.Add(2)

	go func() {
		defer wg.Done()
		bindOutput(cmd.Host, cmd.Path, output, &cmd.Stdout, "", 0)
	}()
	go func() {
		defer wg.Done()
		bindOutput(cmd.Host, cmd.Path, output, &cmd.Stderr, "Error:", console.TextRed)
	}()

	if err = session.Start(cmd.Script); err != nil {
		panic(fmt.Sprintf("[%s] failed to execute command: %s", cmd.Host, err))
		return err
	}

	if err = session.Wait(); err != nil {
		panic(fmt.Sprintf("[%s] failed to wait command: %s", cmd.Host, err))
		return err
	}
	wg.Wait()
	return nil
}

// bing the pipe output for formatted output to channel
func bindOutput(host string, path string, output chan Message, input *io.Reader, prefix string, color int) {
	reader := bufio.NewReader(*input)
	for {
		line, err := reader.ReadString('\n')
		if err != nil || io.EOF == err {
			if err != io.EOF {
				panic(fmt.Sprintf("[%s] faield to execute command: %s", host, err))
			}
			break
		}

		line = prefix + line
		if color != 0 {
			line = console.ColorfulText(color, line)
		}

		output <- Message{
			Host:    host,
			Path:    path,
			Content: line,
		}
	}
}
