package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"url/helper"
	"url/link"
	"url/repository"
)

type Manager struct {
	repo  repository.LinkRepository
	mu    sync.RWMutex
	links map[string]*link.Link
}

func NewManager(repository *repository.LinkRepository) *Manager {
	return &Manager{
		repo:  *repository,
		links: make(map[string]*link.Link),
	}
}

func (m *Manager) CreateShortLink(
	w http.ResponseWriter,
	r *http.Request,
) {

	var request struct {
		URL   string `json:"url"`
		Alias string `json:"alias"`
		TTL   string `json:"ttl"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Println("Не удалось прочитать тело запроса", err)
		helper.WriteJson(w, map[string]string{"Error": "Не удалось прочитать тело запроса"}, http.StatusBadRequest)
		return
	}

	validURL, err := helper.ValidateURL(request.URL)
	if err != nil {
		log.Println("Не корректный URL", err)
		helper.WriteJson(w, map[string]string{"error": "Не корректный URL"}, http.StatusBadRequest)
		return
	}

	var expiresAt *time.Time

	if request.TTL != "" {
		ttl, err := time.ParseDuration(request.TTL)
		if err != nil {
			log.Println("Не корректный TTL")
			helper.WriteJson(w, map[string]string{
				"error": "Не корректный TTL",
			}, http.StatusBadRequest)
			return
		}

		if ttl <= 0 {
			helper.WriteJson(w,
				map[string]string{
					"error": "TTL должен быть больше нуля",
				}, http.StatusBadRequest)

			return
		}

		expiretion := time.Now().Add(ttl)
		expiresAt = &expiretion

	}

	request.URL = validURL
	var code string
	m.mu.Lock()

	if request.Alias != "" {
		alias, err := helper.ValidateAlias(request.Alias)
		if err != nil {
			m.mu.Unlock()

			log.Println("Не корректный alias:", err)

			helper.WriteJson(w, map[string]string{"error": err.Error()}, http.StatusBadRequest)
			return
		}
		_, exists := m.links[alias]
		if exists {
			m.mu.Unlock()

			log.Println("Alias уже занят")

			helper.WriteJson(w, map[string]string{"error": "alias уже занят"}, http.StatusConflict)
			return
		}
		code = alias
	} else {
		generateCode, err := GenerationUniqueCode(m)
		if err != nil {
			m.mu.Unlock()
			log.Println("Не удалось сгенерировать код", err)
			helper.WriteJson(w, map[string]string{"error": "Не удалось сгенерировать код"}, http.StatusInternalServerError)
			return
		}

		code = generateCode

	}
	newLink := &link.Link{
		Code:        code,
		OriginalURL: request.URL,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
	}

	err = m.repo.Create(r.Context(), newLink)
	if err != nil {
		m.mu.Unlock()
		log.Println(err)
		helper.WriteJson(w, map[string]string{
			"error": "не удалось добавить ссылку в базу данных",
		}, http.StatusBadRequest)
		return
	}

	m.links[code] = newLink

	m.mu.Unlock()

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
		newLink, err := m.repo.GetByCode(r.Context(), code)
		if err != nil {
			m.mu.Unlock()
			helper.WriteJson(w, map[string]string{"error": "Код не найден!"}, http.StatusNotFound)
			return
		}
		link = newLink
	}

	if link.ExpiresAt != nil &&
		time.Now().After(*link.ExpiresAt) {

		m.mu.Unlock()

		helper.WriteJson(w, map[string]string{"error": "Срок дейсвтия ссылки истек"}, http.StatusGone)

		return
	}

	link.Clicks++

	err := m.repo.UpdateLink(r.Context(), code)
	if err != nil {
		log.Println("Не удалось обновить ссылку:", err)
	}

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

	m.mu.RLock()
	statlink, ok := m.links[code]
	if !ok {
		m.mu.RUnlock()

		stats, err := m.repo.GetByCode(r.Context(), code)
		if err == nil {
			helper.WriteJson(w, stats, http.StatusOK)
			return
		}
		log.Println("Не найден код")
		helper.WriteJson(w, map[string]string{"error": "Не найден код"}, http.StatusNotFound)
		return
	}

	stats := *statlink

	m.mu.RUnlock()

	helper.WriteJson(w, stats, http.StatusOK)

}

func (m *Manager) DeleteLink(w http.ResponseWriter, r *http.Request) {

	code := r.PathValue("code")

	m.mu.Lock()

	_, exists := m.links[code]
	if !exists {
		m.mu.Unlock()

		err := m.repo.DeleteLinkByCode(r.Context(), code)
		if err == nil {
			helper.WriteJson(w,
				map[string]string{"error": "Ссылка удалена"},
				http.StatusOK)
		}
		helper.WriteJson(w,
			map[string]string{"error": "Сылка не найдена"},
			http.StatusNotFound)

		return
	}

	delete(m.links, code)

	m.mu.Unlock()

	helper.WriteJson(w,
		map[string]string{"error": "Ссылка удалена"},
		http.StatusOK)

}

func (m *Manager) GetAllLinks(w http.ResponseWriter, r *http.Request) {

	links, err := m.repo.GetAllLInks(r.Context())
	if err != nil {
		log.Println("Ошибка:", err)
		helper.WriteJson(w, map[string]string{"error": "Не удалось пполучить ссылки"}, http.StatusBadRequest)
		return
	}

	helper.WriteJson(w, links, http.StatusOK)
}
