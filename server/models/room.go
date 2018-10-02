package models

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

//Room Модель чат комнаты
type Room struct {
	//Канал обмена сообщениями
	MessageChannel chan []byte
	//Текущее количество пользователей
	UsersCount int64
	//Максимальное количество пользователей в чате
	UsersLimit int64
	//Время после которого пользователь считается неактивным
	UserTTL time.Duration
	//Задержка между итерациями проверки неактивных пользователей
	UserActivityMonitorDelay time.Duration
	//Максимальное количество подключений от одного пользователя
	ConnPerUserLimit int64
	//Пользователи - ключ ID - значение ссылка на пользователя
	Users map[string]*User
}

//HandleConnection метод для обработки подключения пользователя
func (room *Room) HandleConnection(conn net.Conn) {
	if room.UsersCount >= room.UsersLimit {
		conn.Write([]byte("Извините, сейчас все места заняты\n"))
		conn.Close()
		return
	}

	user, err := room.authorization(conn)
	if user == nil || err != nil {
		conn.Write([]byte("Невозможно подключиться к серверу\n"))
		conn.Close()
		return
	}

	if user.Connections != nil && int64(len(user.Connections)+1) > room.ConnPerUserLimit {
		conn.Write([]byte("Нельзя подключаться более чем с " + strconv.FormatInt(room.ConnPerUserLimit, 10) + " устройств\n"))
		conn.Close()
		return
	}

	user.AddConnection(conn)
	user.updateLastActionTimestamp()
	_, exists := room.Users[string(user.ID)]
	defer room.closeUserConnection(user, conn)
	//добавляем к основному списку пользователей после авторизации
	//для того чтобы он не видел сообщения до этого момента
	if !exists {
		room.incUsersCount()
		room.Users[string(user.ID)] = user
		room.introduction(user)
	}

	room.messaging(user, conn)
}

//Listen Запуск сервисов комнаты
func (room *Room) Listen(listener *net.TCPListener) {
	room.RunServices()
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		go room.HandleConnection(conn)
	}
}

//RunServices Запуск рассылки и слежения за пользователем
func (room *Room) RunServices() {
	go room.runBroadcasting()
	go room.watchUserActivity()
}

//запуск цикла рассылки
func (room *Room) runBroadcasting() {
	for {
		select {
		case message := <-room.MessageChannel:
			fmt.Println(string(message))
			for _, user := range room.Users {
				if user == nil || user.Connections == nil {
					continue
				}

				for _, conn := range user.Connections {
					conn.Write(message)
				}
			}
		}
	}
}

//запуск слежения за пользователями и удаление неактивных
func (room *Room) watchUserActivity() {
	for {
		for _, user := range room.Users {
			if user.LastActionAt != 0 && user.LastActionAt < (time.Now().Add(-room.UserTTL).Unix()) { // если пользователь был неактивен n-ое время - удалим его
				for _, conn := range user.Connections {
					conn.Write([]byte("Вы были удалены за неактивность\n"))
				}
				room.deleteUserFromRoom(user)
				room.MessageChannel <- append([]byte("Удален неактивный "), append(user.Nickname, byte('\n'))...)
			}
		}

		time.Sleep(room.UserActivityMonitorDelay)
	}
}

//authorization Подключение пользователя к чату
//возвращает статус входа
func (room *Room) authorization(conn net.Conn) (*User, error) {
	conn.Write([]byte("Ваш ник:\n"))
	var buf [128]byte
	for {
		n, err := conn.Read(buf[0:])
		if err != nil {
			return nil, err
		}

		nickname := strings.TrimRight(string(buf[0:n]), "\r\n ")
		validationError := nicknameValidation(nickname)
		if validationError != "" {
			conn.Write([]byte(validationError))
			continue
		}

		return room.getOrCreateUserByNickname(nickname, conn)
	}
}

//introduction Представление пользователя в чате
func (room *Room) introduction(user *User) {
	room.MessageChannel <- append([]byte("К чату присоединился "),
		append(user.Nickname, byte('\n'))...)
}

//messaging Обработка участия пользователя в активности в чате
func (room *Room) messaging(user *User, conn net.Conn) {
	var buf [512]byte
	for {
		resp := bytes.NewBuffer(nil)
		for {
			n, err := conn.Read(buf[0:])
			if err != nil {
				return
			}

			resp.Write(buf[0:n])
			content := resp.Bytes()
			l := resp.Len()
			if l > 1 && string(content[l-1:l]) == "\n" {
				user.updateLastActionTimestamp()
				room.MessageChannel <- append(append(user.Nickname, byte(':')), buf[0:n]...)
			}
		}
	}
}

func (room *Room) incUsersCount() {
	atomic.AddInt64(&room.UsersCount, 1)
}

func (room *Room) decUsersCount() {
	atomic.AddInt64(&room.UsersCount, -1)
}

//Удаление пользователя из чата
func (room *Room) closeUserConnection(user *User, conn net.Conn) {
	user.DeleteConnection(conn)
	//если пользователь разорвал все соединения - удалим его из чата
	if len(user.Connections) == 0 {
		room.deleteUserFromRoom(user)
		room.decUsersCount()
		room.MessageChannel <- append([]byte("Нас покидает "), user.Nickname...)
	}
}

//Удаление пользователя из чата
func (room *Room) deleteUserFromRoom(user *User) {
	//закрываем все соединения
	for _, conn := range user.Connections {
		conn.Close()
	}

	//удаляем из чата
	id := string(user.ID)
	if _, active := room.Users[id]; active {
		delete(room.Users, id)
	}
}

//Создание или получение существующего пользователя
func (room *Room) getOrCreateUserByNickname(nickname string, conn net.Conn) (*User, error) {
	user := room.getUserByNickname(nickname)
	if user != nil {
		return user, nil
	}

	id, err := room.generateUserID()
	if err != nil {
		return nil, err
	}

	return &User{
		ID:       id,
		Nickname: []byte(nickname),
	}, nil
}

//getUserByNickname - метод для поиска пользователя по никнейму
func (room *Room) getUserByNickname(nickname string) *User {
	for _, user := range room.Users {
		if user == nil {
			continue
		}

		if strings.EqualFold(nickname, string(user.Nickname)) {
			return user
		}
	}

	return nil
}

//Генерация уникального ключа пользователя
func (room *Room) generateUserID() ([]byte, error) {
	i := 3
	for i > 0 {
		id, err := generateRandomBytes(32)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}

		unique := true
		for _, user := range room.Users {
			if user == nil {
				continue
			}

			if equalByteSlice(user.ID, id) {
				unique = false
				break
			}
		}

		if unique {
			return id, nil
		}

		i--
	}

	return nil, errors.New("Невозможно создать уникальный ключ")
}
