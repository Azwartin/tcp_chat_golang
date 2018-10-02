package chat

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

const (
	//ui команды
	uiQuit = "\\quit"
)

//Start метод для запуска чата
//Управляет входными, выходными потоками данных
func Start(conn net.Conn) error {
	reader := bufio.NewReader(os.Stdin)
	pipe := make(chan error)
	go listen(conn, pipe)
	for {
		select {
		case err := <-pipe:
			{
				if err != nil {
					return err
				}
			}
		default:
			{
				message, err := reader.ReadString('\n')
				if err != nil {
					return err
				}

				if handleCommand(message, conn) {
					continue
				}

				err = handleMessage(message, conn)
				if err != nil {
					return err
				}
			}
		}

	}
}

//читает соединение и выводит сообщение пользователю
//при возникновении ошибки транслирует ее в основной поток
func listen(conn net.Conn, pipe chan error) {
	var buf [512]byte
	resp := bytes.NewBuffer(nil)
	for {
		n, err := conn.Read(buf[0:])
		if err != nil {
			if err == io.EOF {
				pipe <- err
				return
			}
		}

		resp.Write(buf[0:n])
		content := resp.Bytes()
		len := resp.Len()
		if string(content[len-1:len]) == "\n" {
			fmt.Print(string(content))
			resp.Reset()
		}
	}
}

//отправка сообщения
func handleMessage(message string, conn net.Conn) error {
	message = strings.TrimRight(message, "\t\r\n")
	if len(message) == 0 {
		return nil
	}

	_, err := conn.Write(append([]byte(message), byte('\n')))
	return err
}

//перехватыват и обработка команды из общего потока ввода
//возвращает true если была обработана команда и false в противном случае
func handleCommand(str string, conn net.Conn) bool {
	switch strings.ToLower(strings.TrimRight(str, "\n")) {
	case strings.ToLower(uiQuit):
		{
			conn.Close()
			os.Exit(0)
		}
	}

	return false
}
