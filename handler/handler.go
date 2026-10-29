package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"url/helper"
	"url/link"
	"url/repository"

	"github.com/jackc/pgx/v5"
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

		if errors.Is(err, repository.ErrCodeTaken) {

			if request.Alias != "" {
				m.mu.Unlock()

				helper.WriteJson(w,
					map[string]string{
						"error": "Алиас уже занят",
					},
					http.StatusConflict)
				return
			}

			for attempt := 0; attempt < 3; attempt++ {

				code, err := GenerationUniqueCode(m)

				if err != nil {

					m.mu.Unlock()

					log.Println(err)

					helper.WriteJson(w,
						map[string]string{
							"error": "Не удалось сгенерировать код",
						}, http.StatusInternalServerError)

					return
				}

				newLink.Code = code

				err = m.repo.Create(r.Context(), newLink)
				if err != nil {
					if errors.Is(err, repository.ErrCodeTaken) {
						continue
					}

					log.Println("Не удалось создать ссылку:", err)
					m.mu.Unlock()

					helper.WriteJson(w, map[string]string{
						"error": "не удалось добавить ссылку в базу данных"},
						http.StatusInternalServerError)

					return
				}

				m.links[code] = newLink

				response := map[string]string{
					"short_url": "http://localhost:8080/r/" + code,
				}

				helper.WriteJson(w, response, http.StatusCreated)

				m.mu.Unlock()
				return
			}
			helper.WriteJson(w,
				map[string]string{"error": "Не удалось создать короткую ссылку"},
				http.StatusInternalServerError)

			m.mu.Unlock()
			return

		}

		log.Println("Не удалось создать ссылку:", err)

		m.mu.Unlock()

		helper.WriteJson(w, map[string]string{
			"error": "не удалось добавить ссылку в базу данных"},
			http.StatusInternalServerError)

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

	newLink, err := m.repo.GetAndIncrement(r.Context(), code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {

			helper.WriteJson(w,
				map[string]string{"error": "Ссылка не найдена"},
				http.StatusNotFound)

			return

		}

		log.Printf("Не удалось получить ссылку с кодом %q: %v", code, err)

		helper.WriteJson(w,
			map[string]string{"error": "Не удалось получить ссылку"},
			http.StatusInternalServerError)

		return
	}

	if newLink.ExpiresAt != nil &&
		time.Now().After(*newLink.ExpiresAt) {

		helper.WriteJson(w,
			map[string]string{"error": "Срок дейсвтия ссылки истек"},
			http.StatusGone)

		return
	}

	originalURL := newLink.OriginalURL

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

	err := m.repo.DeleteLinkByCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			helper.WriteJson(w,
				map[string]string{"error": "Not found link"},
				http.StatusNotFound)
			return
		}

		log.Println("error delete link:", err)
		helper.WriteJson(w,
			map[string]string{"error": "error delete link"},
			http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
