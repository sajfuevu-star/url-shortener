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
		helper.WriteJson(w, err, http.StatusBadRequest)
		return
	}

	request.URL = url

	code, err := GenerationUniqueCode(m)
	if err != nil {
		log.Println("Не удалось сгенерировать код", err)
		helper.WriteJson(w, map[string]string{"error":"Не удалось сгенерировать код"}, http.StatusBadRequest)
		return
	}

	link := &link.Link{
		Code:        code,
		OriginalURL: request.URL,
	}

	m.mu.Lock()
	m.links[code] = link
	m.mu.Unlock()

	response := map[string]string{
		"short_url": "http://localhost:8080/r/" + code,
	}

	helper.WriteJson(w, response, http.StatusCreated)

}


func (m *Manager) GetShrotLink (w http.ResponseWriter, r *http.Request) {
	var request struct {
		URL  string   `json:"url"`
	}

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Println("Не удалось прочитать тело запроса!")
		return
	}

	for i := range m.links {
		if m.links[i].OriginalURL == request.URL{
			m.Redirect(w,r)
		}
	}


 
}

func (m *Manager) Redirect(w http.ResponseWriter, r *http.Request){

	code := r.PathValue("code")

	link, ok := m.links[code]
	if !ok {
		helper.WriteJson(w,map[string]string{"error":"Код не найден!"}, http.StatusNotFound)
		return
	}

	link.Clicks++

	http.Redirect(w,r,link.OriginalURL,http.StatusFound)
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
