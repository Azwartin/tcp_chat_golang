package models

import (
	"crypto/rand"
	"strings"
)

//Метод для генерации случайного массива byte
func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

//Метод для проверки эквивалентности двух байтовых слайсов
func equalByteSlice(a, b []byte) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return len(a) == len(b)
}

//Метод для проверки валидности никнейма
//При ошибке валидации возвращает строку с описанием ошибки
//Пустая строка - валидация пройдена
func nicknameValidation(nickname string) string {
	nickname = strings.Trim(nickname, " ")
	if len(nickname) < 3 {
		return "Длинна никнейма должна быть больше 3-х символов\n"
	}

	if len(nickname) > 30 {
		return "Длинна никнейма должна быть меньше 30-х символов\n"
	}

	return ""
}
