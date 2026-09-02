package helper

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strings"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func WriteJson(w http.ResponseWriter, msg any, status int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(msg); err != nil {
		log.Println("Не удалось отправить JSON")
		return err
	}

	return nil
}

func GenerationCode(lenght int) (string, error) {
	code := make([]byte, lenght)

	for i := range code {
		randomIndex, err := rand.Int(
			rand.Reader,
			big.NewInt(int64(len(alphabet))),
		)
		if err != nil {
			return "", err
		}

		code[i] = alphabet[randomIndex.Int64()]
	}
	return string(code), nil
}

func ValidateURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)

	if rawURL == "" {
		return "", fmt.Errorf("URL не указан")
	}

	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return "", fmt.Errorf("некорректный URL")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("разрешены только протоколы http и https")
	}

	if parsedURL.Host == "" {
		return "", fmt.Errorf("URL должен содержать домен")
	}

	return rawURL, nil
}
