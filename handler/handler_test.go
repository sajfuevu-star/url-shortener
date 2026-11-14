package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"url/repository"
)

func TestCreateShortLink_InvalidJSON(t *testing.T) {

	body := strings.NewReader(`{"url":`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/shorten",
		body,
	)

	recover := httptest.NewRecorder()

	manager := &Manager{}

	manager.CreateShortLink(recover, req)

	if recover.Code != http.StatusBadRequest {
		t.Errorf(
			"Ожидали статус 400, получили %d",
			recover.Code,
		)
		return
	}
}

func TestCreateShortLink_InvalidURL(t *testing.T) {

	var response struct {
		Error string `json:"error"`
	}

	body := strings.NewReader(`{"url":"hfhh"}`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/shorten",
		body,
	)

	recorder := httptest.NewRecorder()

	manager := &Manager{}

	manager.CreateShortLink(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf(
			"Ожидали статус 400, получили %d",
			recorder.Code,
		)
	}

	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Не удалось распознать JSON: %v", err)
	}

	if response.Error != "Не корректный URL" {
		t.Errorf("Ожидали не корректный URL плучили %v", response.Error)
	}

}

type MockRepository struct {
	CreateErr   error
	CreateCalls int
}

func (m *MockRepository) Create(ctx context.Context, newLink *repository.LinkRepository) error {
	m.CreateCalls++
	return m.CreateErr
}
