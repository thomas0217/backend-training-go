package form_test

import (
	"awesomeProject/internal/form"
	"awesomeProject/internal/form/mocks"
	"awesomeProject/internal/jwt"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

func newHandler(t *testing.T) (*mocks.Store, *form.Handler) {
	t.Helper()
	store := mocks.NewStore(t)
	logger := zaptest.NewLogger(t)
	v := validator.New()
	return store, form.NewHandler(logger, v, store)
}

func TestHandler_Create(t *testing.T) {
	tests := []struct {
		name         string
		formID       uuid.UUID
		userID       uuid.UUID    // ID of the user making the request
		reqBody      form.Request // Customize based on actual request structure
		customBody   []byte       // Optional raw body for more complex cases
		setMock      func(store *mocks.Store, formID uuid.UUID)
		expectStatus int
	}{
		{
			name:   "Successful form creation",
			formID: uuid.New(),
			userID: uuid.New(),
			reqBody: form.Request{
				Title:       "Test Form",
				Description: "This is a test form",
			},
			setMock: func(store *mocks.Store, formID uuid.UUID) {
				store.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(form.Form{
					ID:          formID,
					Title:       "Test Form",
					Description: pgtype.Text{String: "This is a test form", Valid: true},
				}, nil)
			},
			expectStatus: 201,
		},
		{
			name:   "Missing title",
			formID: uuid.New(),
			userID: uuid.New(),
			reqBody: form.Request{
				Description: "This is a test form without title",
			},
			setMock:      func(store *mocks.Store, formID uuid.UUID) {},
			expectStatus: 400,
		},
		{
			name:   "Database error on creation",
			formID: uuid.New(),
			userID: uuid.New(),
			reqBody: form.Request{
				Title:       "Test Form",
				Description: "This is a test form",
			},
			setMock: func(store *mocks.Store, formID uuid.UUID) {
				store.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(form.Form{}, errors.New("database error"))
			},
			expectStatus: 500,
		},
		{
			name:   "Invalid description type",
			formID: uuid.New(),
			userID: uuid.New(),
			customBody: []byte(`{
				"title": "Test Form",
				"description": 12345
			}`),
			setMock:      func(store *mocks.Store, formID uuid.UUID) {},
			expectStatus: 400,
		},
		{
			name:         "Empty request body",
			formID:       uuid.New(),
			userID:       uuid.New(),
			customBody:   []byte(`{}`),
			setMock:      func(store *mocks.Store, formID uuid.UUID) {},
			expectStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, handler := newHandler(t)
			tt.setMock(store, tt.formID)
			var rawBody []byte
			if tt.customBody != nil {
				rawBody = tt.customBody
			} else {
				rawBody, _ = json.Marshal(tt.reqBody)
			}

			r := httptest.NewRequest(http.MethodPost, "/api/forms", bytes.NewBuffer(rawBody))
			w := httptest.NewRecorder()

			r = r.WithContext(context.WithValue(r.Context(), jwt.UserContextKey, tt.userID))
			handler.Create(w, r)

			assert.Equalf(t, tt.expectStatus, w.Result().StatusCode, "Expected status code to match, Expected %d, got %d", tt.expectStatus, w.Result().StatusCode)
		})
	}
}

func TestHandler_List(t *testing.T) {
	tests := []struct {
		name         string
		userID       uuid.UUID
		withUserCtx  bool
		setMock      func(store *mocks.Store, userID uuid.UUID)
		expectStatus int
	}{
		{
			name:        "Successful list",
			userID:      uuid.New(),
			withUserCtx: true,
			setMock: func(store *mocks.Store, userID uuid.UUID) {
				store.On("List", mock.Anything, userID).Return([]form.ListRow{
					{
						ID:          uuid.New(),
						Title:       "Form 1",
						Description: pgtype.Text{String: "Desc 1", Valid: true},
						IsBookmark:  false,
					},
					{
						ID:          uuid.New(),
						Title:       "Form 2",
						Description: pgtype.Text{String: "Desc 2", Valid: true},
						IsBookmark:  true,
					},
				}, nil)
			},
			expectStatus: 200,
		},
		{
			name:        "Empty list",
			userID:      uuid.New(),
			withUserCtx: true,
			setMock: func(store *mocks.Store, userID uuid.UUID) {
				store.On("List", mock.Anything, userID).Return([]form.ListRow{}, nil)
			},
			expectStatus: 200,
		},
		{
			name:         "Missing user context",
			userID:       uuid.UUID{},
			withUserCtx:  false,
			setMock:      func(store *mocks.Store, userID uuid.UUID) {},
			expectStatus: 401,
		},
		{
			name:        "Database error on list",
			userID:      uuid.New(),
			withUserCtx: true,
			setMock: func(store *mocks.Store, userID uuid.UUID) {
				store.On("List", mock.Anything, userID).Return(nil, errors.New("database error"))
			},
			expectStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, handler := newHandler(t)
			tt.setMock(store, tt.userID)

			r := httptest.NewRequest(http.MethodGet, "/api/forms", nil)
			w := httptest.NewRecorder()

			if tt.withUserCtx {
				r = r.WithContext(context.WithValue(r.Context(), jwt.UserContextKey, tt.userID))
			}
			handler.List(w, r)

			assert.Equalf(t, tt.expectStatus, w.Result().StatusCode, "Expected status code %d, got %d", tt.expectStatus, w.Result().StatusCode)
		})
	}
}

func TestHandler_Update(t *testing.T) {
	tests := []struct {
		name         string
		formID       uuid.UUID
		pathID       string
		reqBody      form.Request
		customBody   []byte
		setMock      func(store *mocks.Store, formID uuid.UUID)
		expectStatus int
	}{
		{
			name:   "Successful update",
			formID: uuid.New(),
			reqBody: form.Request{
				Title:       "Updated Title",
				Description: "Updated Description",
			},
			setMock: func(store *mocks.Store, formID uuid.UUID) {
				store.On("Update", mock.Anything, formID, "Updated Title", "Updated Description").Return(form.Form{
					ID:          formID,
					Title:       "Updated Title",
					Description: pgtype.Text{String: "Updated Description", Valid: true},
				}, nil)
			},
			expectStatus: 200,
		},
		{
			name:         "Invalid form ID",
			formID:       uuid.UUID{},
			pathID:       "not-a-uuid",
			reqBody:      form.Request{},
			setMock:      func(store *mocks.Store, formID uuid.UUID) {},
			expectStatus: 400,
		},
		{
			name:   "Invalid request body",
			formID: uuid.New(),
			customBody: []byte(`{
				"title": 12345,
				"description": "desc"
			}`),
			setMock:      func(store *mocks.Store, formID uuid.UUID) {},
			expectStatus: 400,
		},
		{
			name:   "Database error on update",
			formID: uuid.New(),
			reqBody: form.Request{
				Title:       "Title",
				Description: "Desc",
			},
			setMock: func(store *mocks.Store, formID uuid.UUID) {
				store.On("Update", mock.Anything, formID, "Title", "Desc").Return(form.Form{}, errors.New("database error"))
			},
			expectStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, handler := newHandler(t)
			tt.setMock(store, tt.formID)

			var rawBody []byte
			if tt.customBody != nil {
				rawBody = tt.customBody
			} else {
				rawBody, _ = json.Marshal(tt.reqBody)
			}

			pathID := tt.pathID
			if pathID == "" {
				pathID = tt.formID.String()
			}

			r := httptest.NewRequest(http.MethodPut, "/api/forms/"+pathID, bytes.NewBuffer(rawBody))
			r.SetPathValue("id", pathID)
			w := httptest.NewRecorder()

			handler.Update(w, r)

			assert.Equalf(t, tt.expectStatus, w.Result().StatusCode, "Expected status code %d, got %d", tt.expectStatus, w.Result().StatusCode)
		})
	}
}

func TestHandler_Delete(t *testing.T) {
	tests := []struct {
		name         string
		formID       uuid.UUID
		pathID       string
		setMock      func(store *mocks.Store, formID uuid.UUID)
		expectStatus int
	}{
		{
			name:   "Successful delete",
			formID: uuid.New(),
			setMock: func(store *mocks.Store, formID uuid.UUID) {
				store.On("Delete", mock.Anything, formID).Return(nil)
			},
			expectStatus: 204,
		},
		{
			name:         "Invalid form ID",
			formID:       uuid.UUID{},
			pathID:       "not-a-uuid",
			setMock:      func(store *mocks.Store, formID uuid.UUID) {},
			expectStatus: 400,
		},
		{
			name:   "Database error on delete",
			formID: uuid.New(),
			setMock: func(store *mocks.Store, formID uuid.UUID) {
				store.On("Delete", mock.Anything, formID).Return(errors.New("database error"))
			},
			expectStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, handler := newHandler(t)
			tt.setMock(store, tt.formID)

			pathID := tt.pathID
			if pathID == "" {
				pathID = tt.formID.String()
			}

			r := httptest.NewRequest(http.MethodDelete, "/api/forms/"+pathID, nil)
			r.SetPathValue("id", pathID)
			w := httptest.NewRecorder()

			handler.Delete(w, r)

			assert.Equalf(t, tt.expectStatus, w.Result().StatusCode, "Expected status code %d, got %d", tt.expectStatus, w.Result().StatusCode)
		})
	}
}
