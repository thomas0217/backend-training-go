package bookmark_test

import (
	"awesomeProject/internal/bookmark"
	"awesomeProject/internal/bookmark/mocks"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestService_ToggleBookmark(t *testing.T) {
	userID := uuid.New()
	formID := uuid.New()

	tests := []struct {
		name           string
		setMock        func(q *mocks.Query)
		expectBookmark bool
		expectErr      bool
	}{
		{
			name: "bookmark does not exist - should add",
			setMock: func(q *mocks.Query) {
				q.On("Exist", context.Background(), bookmark.ExistParams{UserID: userID, FormID: formID}).
					Return(false, nil)
				q.On("Create", context.Background(), bookmark.CreateParams{UserID: userID, FormID: formID}).
					Return(bookmark.Bookmark{}, nil)
			},
			expectBookmark: true,
			expectErr:      false,
		},
		{
			name: "bookmark exists - should remove",
			setMock: func(q *mocks.Query) {
				q.On("Exist", context.Background(), bookmark.ExistParams{UserID: userID, FormID: formID}).
					Return(true, nil)
				q.On("Delete", context.Background(), bookmark.DeleteParams{UserID: userID, FormID: formID}).
					Return(nil)
			},
			expectBookmark: false,
			expectErr:      false,
		},
		{
			name: "exist check fails",
			setMock: func(q *mocks.Query) {
				q.On("Exist", context.Background(), bookmark.ExistParams{UserID: userID, FormID: formID}).
					Return(false, errors.New("database error"))
			},
			expectBookmark: false,
			expectErr:      true,
		},
		{
			name: "delete fails",
			setMock: func(q *mocks.Query) {
				q.On("Exist", context.Background(), bookmark.ExistParams{UserID: userID, FormID: formID}).
					Return(true, nil)
				q.On("Delete", context.Background(), bookmark.DeleteParams{UserID: userID, FormID: formID}).
					Return(errors.New("database error"))
			},
			expectBookmark: false,
			expectErr:      true,
		},
		{
			name: "create fails",
			setMock: func(q *mocks.Query) {
				q.On("Exist", context.Background(), bookmark.ExistParams{UserID: userID, FormID: formID}).
					Return(false, nil)
				q.On("Create", context.Background(), bookmark.CreateParams{UserID: userID, FormID: formID}).
					Return(bookmark.Bookmark{}, errors.New("database error"))
			},
			expectBookmark: false,
			expectErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			logger := zaptest.NewLogger(t)
			q := mocks.NewQuery(t)
			tt.setMock(q)
			svc := bookmark.NewServiceWithQuery(logger, q)

			// Act
			bookmarked, err := svc.ToggleBookmark(context.Background(), userID, formID)

			// Assert
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectBookmark, bookmarked)
		})
	}
}
