package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"url/helper"
	"url/link"
)

type Manager struct {
	mu    sync.RWMutex
	links map[string]*link.Link
}

func NewManager() *Manager {
	return &Manager{
		links: make(map[string]*link.Link),
	}
}

func (m *Manager) CreateShortLink(
	w http.ResponseWriter,
	r *http.Request,
) {

	var request struct {
		URL string `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Println("Не удалось прочитать тело запроса", err)
		helper.WriteJson(w, map[string]string{"Error": "Не удалось прочитать тело запроса"}, http.StatusBadRequest)
		return
	}

	url, err := helper.ValidateURL(request.URL)
	if err != nil {
		log.Println("Не корректный URL", err)
		helper.WriteJson(w, map[string]string{"error": "Не корректный URL"}, http.StatusBadRequest)
		return
	}

	request.URL = url
	m.mu.Lock()
	defer m.mu.Unlock()

	code, err := GenerationUniqueCode(m)
	if err != nil {
		log.Println("Не удалось сгенерировать код", err)
		helper.WriteJson(w, map[string]string{"error": "Не удалось сгенерировать код"}, http.StatusInternalServerError)
		return
	}

	link := &link.Link{
		Code:        code,
		OriginalURL: request.URL,
	}

	m.links[code] = link

	response := map[string]string{
		"short_url": "http://localhost:8080/r/" + code,
	}

	helper.WriteJson(w, response, http.StatusCreated)

}

func (m *Manager) Redirect(w http.ResponseWriter, r *http.Request) {

	code := r.PathValue("code")

	m.mu.Lock()

	link, ok := m.links[code]
	if !ok {
		m.mu.Unlock()

		helper.WriteJson(w, map[string]string{"error": "Код не найден!"}, http.StatusNotFound)
		return
	}

	link.Clicks++

	originalURL := link.OriginalURL
	m.mu.Unlock()

	http.Redirect(w, r, originalURL, http.StatusFound)
}

func GenerationUniqueCode(m *Manager) (string, error) {
	for attempt := 0; attempt < 10; attempt++ {
		code, err := helper.GenerationCode(6)
		if err != nil {
			return "", err
		}
		_, exists := m.links[code]
		if !exists {
			return code, nil
		}
	}

	return "", fmt.Errorf("не удалось создать код")
}

func (m *Manager) GetStats(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	m.mu.Lock()
	statlink, ok := m.links[code]
	if !ok {
		m.mu.Unlock()

		log.Println("Не найден код")
		helper.WriteJson(w, map[string]string{"error": "Не найден код"}, http.StatusNotFound)
		return
	}

	stats := *statlink

	m.mu.Unlock()

	helper.WriteJson(w, stats, http.StatusOK)

}
