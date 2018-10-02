package models

import (
	"net"
	"time"
)

//User Модель пользователя
type User struct {
	//уникальный идентификатор пользователя
	ID []byte
	//подключения пользователя
	Connections []net.Conn
	//nickname
	Nickname []byte
	//timestamp последнего действия
	LastActionAt int64
}

//AddConnection Добавление соединения
func (user *User) AddConnection(conn net.Conn) {
	if user.Connections == nil {
		user.Connections = []net.Conn{conn}
	} else {
		user.Connections = append(user.Connections, conn)
	}
}

//DeleteConnection Удаление соединения пользователя
func (user *User) DeleteConnection(conn net.Conn) {
	for i, cn := range user.Connections {
		if cn == conn {
			user.Connections = append(user.Connections[0:i], user.Connections[i+1:]...)
		}
	}

	conn.Close()
}

func (user *User) updateLastActionTimestamp() {
	user.LastActionAt = time.Now().Unix()
}
