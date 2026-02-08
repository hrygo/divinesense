package context

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEpisodicProvider is a mock for EpisodicProvider.
type MockEpisodicProvider struct {
	mock.Mock
}

func (m *MockEpisodicProvider) SearchEpisodes(ctx context.Context, userID int32, query string, limit int) ([]*EpisodicMemory, error) {
	args := m.Called(ctx, userID, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*EpisodicMemory), args.Error(1)
}

// MockPreferenceProvider is a mock for PreferenceProvider.
type MockPreferenceProvider struct {
	mock.Mock
}

func (m *MockPreferenceProvider) GetPreferences(ctx context.Context, userID int32) (*UserPreferences, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserPreferences), args.Error(1)
}

func TestLongTermExtractor_New(t *testing.T) {
	t.Run("Default max episodes", func(t *testing.T) {
		extractor := NewLongTermExtractor(0)
		assert.Equal(t, 3, extractor.maxEpisodes)
	})

	t.Run("Custom max episodes", func(t *testing.T) {
		extractor := NewLongTermExtractor(10)
		assert.Equal(t, 10, extractor.maxEpisodes)
	})

	t.Run("Negative max episodes uses default", func(t *testing.T) {
		extractor := NewLongTermExtractor(-5)
		assert.Equal(t, 3, extractor.maxEpisodes)
	})
}

func TestLongTermExtractor_Extract(t *testing.T) {
	ctx := context.Background()
	extractor := NewLongTermExtractor(5)

	t.Run("Successful extraction", func(t *testing.T) {
		mockEpisodic := new(MockEpisodicProvider)
		mockPref := new(MockPreferenceProvider)

		episodes := []*EpisodicMemory{
			{
				ID:        1,
				Timestamp: time.Now().Add(-24 * time.Hour),
				Summary:   "Searched for Go documentation",
				AgentType: "memo",
			},
			{
				ID:        2,
				Timestamp: time.Now().Add(-48 * time.Hour),
				Summary:   "Created meeting notes",
				AgentType: "amazing",
			},
		}

		prefs := &UserPreferences{
			Timezone:           "Asia/Shanghai",
			DefaultDuration:    60,
			PreferredTimes:     []string{"09:00", "14:00"},
			CommunicationStyle: "concise",
		}

		mockEpisodic.On("SearchEpisodes", ctx, int32(1), "test query", 5).
			Return(episodes, nil)
		mockPref.On("GetPreferences", ctx, int32(1)).
			Return(prefs, nil)

		result, err := extractor.Extract(ctx, mockEpisodic, mockPref, 1, "test query")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, episodes, result.Episodes)
		assert.Equal(t, prefs, result.Preferences)

		mockEpisodic.AssertExpectations(t)
		mockPref.AssertExpectations(t)
	})

	t.Run("Nil episodic provider", func(t *testing.T) {
		mockPref := new(MockPreferenceProvider)

		prefs := &UserPreferences{Timezone: "UTC"}

		mockPref.On("GetPreferences", ctx, int32(1)).
			Return(prefs, nil)

		result, err := extractor.Extract(ctx, nil, mockPref, 1, "test query")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Nil(t, result.Episodes)
		assert.Equal(t, prefs, result.Preferences)
	})

	t.Run("Nil preference provider", func(t *testing.T) {
		mockEpisodic := new(MockEpisodicProvider)

		episodes := []*EpisodicMemory{
			{ID: 1, Summary: "Test episode"},
		}

		mockEpisodic.On("SearchEpisodes", ctx, int32(1), "test query", 5).
			Return(episodes, nil)

		result, err := extractor.Extract(ctx, mockEpisodic, nil, 1, "test query")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, episodes, result.Episodes)
		// Should use default preferences
		assert.NotNil(t, result.Preferences)
	})

	t.Run("Episodic provider error - non-fatal", func(t *testing.T) {
		mockEpisodic := new(MockEpisodicProvider)
		mockPref := new(MockPreferenceProvider)

		mockEpisodic.On("SearchEpisodes", ctx, int32(1), "test query", 5).
			Return(nil, assert.AnError)
		mockPref.On("GetPreferences", ctx, int32(1)).
			Return(&UserPreferences{}, nil)

		result, err := extractor.Extract(ctx, mockEpisodic, mockPref, 1, "test query")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Nil(t, result.Episodes) // Error should result in nil episodes
	})

	t.Run("Preference provider error - uses defaults", func(t *testing.T) {
		mockEpisodic := new(MockEpisodicProvider)
		mockPref := new(MockPreferenceProvider)

		mockEpisodic.On("SearchEpisodes", ctx, int32(1), "test query", 5).
			Return([]*EpisodicMemory{}, nil)
		mockPref.On("GetPreferences", ctx, int32(1)).
			Return(nil, assert.AnError)

		result, err := extractor.Extract(ctx, mockEpisodic, mockPref, 1, "test query")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		// Should use default preferences
		assert.NotNil(t, result.Preferences)
		assert.Equal(t, "Asia/Shanghai", result.Preferences.Timezone)
	})
}

func TestFormatEpisodes_LongTerm(t *testing.T) {
	t.Run("Empty episodes", func(t *testing.T) {
		result := FormatEpisodes([]*EpisodicMemory{})
		assert.Empty(t, result)
	})

	t.Run("Single episode", func(t *testing.T) {
		episodes := []*EpisodicMemory{
			{
				Timestamp: time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
				Summary:   "Test summary",
			},
		}

		result := FormatEpisodes(episodes)

		assert.Contains(t, result, "相关历史")
		assert.Contains(t, result, "01-15 10:30")
		assert.Contains(t, result, "Test summary")
	})

	t.Run("Multiple episodes", func(t *testing.T) {
		episodes := []*EpisodicMemory{
			{
				Timestamp: time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
				Summary:   "First episode",
			},
			{
				Timestamp: time.Date(2026, 1, 16, 14, 0, 0, 0, time.UTC),
				Summary:   "Second episode",
			},
		}

		result := FormatEpisodes(episodes)

		assert.Contains(t, result, "First episode")
		assert.Contains(t, result, "Second episode")
	})
}

func TestDefaultUserPreferences(t *testing.T) {
	prefs := DefaultUserPreferences()

	assert.Equal(t, "Asia/Shanghai", prefs.Timezone)
	assert.Equal(t, 60, prefs.DefaultDuration)
	assert.Equal(t, []string{"09:00", "14:00"}, prefs.PreferredTimes)
	assert.Equal(t, "concise", prefs.CommunicationStyle)
}

// Benchmark_Extract benchmarks long-term extraction.
func BenchmarkLongTermExtract(b *testing.B) {
	ctx := context.Background()
	extractor := NewLongTermExtractor(5)
	mockEpisodic := new(MockEpisodicProvider)
	mockPref := new(MockPreferenceProvider)

	episodes := make([]*EpisodicMemory, 10)
	for i := range episodes {
		episodes[i] = &EpisodicMemory{
			ID:        int64(i),
			Timestamp: time.Now(),
			Summary:   "Episode summary",
		}
	}

	mockEpisodic.On("SearchEpisodes", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(episodes, nil)
	mockPref.On("GetPreferences", mock.Anything, mock.Anything).
		Return(DefaultUserPreferences(), nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = extractor.Extract(ctx, mockEpisodic, mockPref, 1, "test")
	}
}
