package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type TaskRepo interface {
	Create(title string) (Task, error)
	Get(id string) (Task, bool)
	List(done bool) []Task
	SetDone(id string, done bool) (Task, error)
}
type Clock interface{ Now() time.Time }

var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidTitle = errors.New("invalid title")
)

var taskIDSequence atomic.Uint64

type inMemoryTaskRepo struct {
	mu    sync.RWMutex
	clock Clock
	tasks map[string]Task
}

func NewInMemoryTaskRepo(clock Clock) TaskRepo {
	return &inMemoryTaskRepo{
		clock: clock,
		tasks: make(map[string]Task),
	}
}

func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, ErrInvalidTitle
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	id := strconv.FormatUint(taskIDSequence.Add(1), 10)
	newTask := Task{
		ID:        id,
		Title:     title,
		Done:      false,
		UpdatedAt: r.clock.Now(),
	}
	r.tasks[id] = newTask
	return newTask, nil
}

func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, ok := r.tasks[id]
	return task, ok
}

func (r *inMemoryTaskRepo) List(done bool) []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Task, 0)
	for _, task := range r.tasks {
		if task.Done == done {
			result = append(result, task)
		}
	}
	return result
}

func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	task, ok := r.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	if task.Done == done {
		return task, nil
	}

	task.Done = done
	task.UpdatedAt = r.clock.Now()

	r.tasks[id] = task

	return task, nil
}

type httpHandler struct{ repo TaskRepo }

func NewHTTPHandler(repo TaskRepo) http.Handler {
	return &httpHandler{repo: repo}
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/tasks":
		switch r.Method {
		case http.MethodGet:
			h.handleList(w, r)

		case http.MethodPost:
			h.handleCreate(w, r)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}

	case strings.HasPrefix(r.URL.Path, "/tasks/"):
		id := strings.TrimPrefix(r.URL.Path, "/tasks/")

		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.handleGet(w, r, id)

		case http.MethodPatch:
			h.handlePatch(w, r, id)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}

	default:
		http.NotFound(w, r)
	}
}

func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Title *string `json:"title"`
	}
	if err := decodeStrictJSON(r.Body, &payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if payload.Title == nil {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}
	task, err := h.repo.Create(*payload.Title)
	switch {
	case err == nil:
		writeJSON(w, http.StatusCreated, task)
	case errors.Is(err, ErrInvalidTitle):
		http.Error(w, "invalid title", http.StatusBadRequest)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	task, ok := h.repo.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	done, err := parseDoneFilter(r.URL.RawQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tasks := h.repo.List(done)

	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].UpdatedAt.Equal(tasks[j].UpdatedAt) {
			return tasks[i].ID < tasks[j].ID
		}

		return tasks[i].UpdatedAt.After(tasks[j].UpdatedAt)
	})

	if tasks == nil {
		tasks = []Task{}
	}

	writeJSON(w, http.StatusOK, tasks)
}

func parseDoneFilter(query string) (bool, error) {
	var done, seen bool
	for _, parameter := range strings.Split(query, "&") {
		key, value, _ := strings.Cut(parameter, "=")
		key, err := url.QueryUnescape(key)
		if err != nil || key != "done" {
			continue
		}
		value, err = url.QueryUnescape(value)
		if seen || err != nil || (value != "true" && value != "false") {
			return false, errors.New("invalid done parameter")
		}
		seen = true
		done = value == "true"
	}
	return done, nil
}

func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var payload struct {
		Done *bool `json:"done"`
	}

	if err := decodeStrictJSON(r.Body, &payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if payload.Done == nil {
		http.Error(w, "done is required", http.StatusBadRequest)
		return
	}

	task, err := h.repo.SetDone(id, *payload.Done)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, task)
	case errors.Is(err, ErrNotFound):
		http.NotFound(w, r)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func decodeStrictJSON(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)

	var raw json.RawMessage

	if err := decoder.Decode(&raw); err != nil {
		return err
	}

	data := bytes.TrimSpace(raw)
	if len(data) == 0 || data[0] != '{' {
		return errors.New("expected JSON object")
	}

	var extra any

	err := decoder.Decode(&extra)

	if err == nil {
		return errors.New("multiple JSON values")
	}
	if !errors.Is(err, io.EOF) {
		return err
	}

	strictDecoder := json.NewDecoder(bytes.NewReader(raw))
	strictDecoder.DisallowUnknownFields()

	return strictDecoder.Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(v)
}
