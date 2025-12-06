package usecase

import (
	"context"
	"testing"

	"smap-collector/internal/results"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLogger implements log.Logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(ctx context.Context, arg ...any)                   {}
func (m *mockLogger) Debugf(ctx context.Context, template string, arg ...any) {}
func (m *mockLogger) Info(ctx context.Context, arg ...any)                    {}
func (m *mockLogger) Infof(ctx context.Context, template string, arg ...any)  {}
func (m *mockLogger) Warn(ctx context.Context, arg ...any)                    {}
func (m *mockLogger) Warnf(ctx context.Context, template string, arg ...any)  {}
func (m *mockLogger) Error(ctx context.Context, arg ...any)                   {}
func (m *mockLogger) Errorf(ctx context.Context, template string, arg ...any) {}
func (m *mockLogger) Fatal(ctx context.Context, arg ...any)                   {}
func (m *mockLogger) Fatalf(ctx context.Context, template string, arg ...any) {}
func (m *mockLogger) SetLevel(level string) error                             { return nil }

// TestMapCrawlerContentDataWithTitle tests mapping ContentData with YouTube title
func TestMapCrawlerContentDataWithTitle(t *testing.T) {
	ctx := context.Background()
	uc := implUseCase{l: &mockLogger{}}

	title := "Test YouTube Video Title"
	crawlerData := results.CrawlerContentData{
		Text:     "Test video description",
		Duration: 120,
		Hashtags: []string{"test", "youtube"},
		Title:    &title,
	}

	projectData, err := uc.mapCrawlerContentDataToProjectContentData(ctx, "job123", "youtube", "video123", crawlerData)

	require.NoError(t, err)
	assert.Equal(t, "Test video description", projectData.Text)
	assert.Equal(t, 120, projectData.Duration)
	assert.NotNil(t, projectData.Title)
	assert.Equal(t, "Test YouTube Video Title", *projectData.Title)
}

// TestMapCrawlerContentDataWithoutTitle tests mapping ContentData without title (TikTok case)
func TestMapCrawlerContentDataWithoutTitle(t *testing.T) {
	ctx := context.Background()
	uc := implUseCase{l: &mockLogger{}}

	crawlerData := results.CrawlerContentData{
		Text:      "Test TikTok video",
		Duration:  60,
		Hashtags:  []string{"tiktok", "test"},
		SoundName: "Original Sound",
		Title:     nil, // TikTok doesn't have title
	}

	projectData, err := uc.mapCrawlerContentDataToProjectContentData(ctx, "job456", "tiktok", "video456", crawlerData)

	require.NoError(t, err)
	assert.Equal(t, "Test TikTok video", projectData.Text)
	assert.Equal(t, 60, projectData.Duration)
	assert.Nil(t, projectData.Title) // Should be nil for TikTok
	assert.Equal(t, "Original Sound", projectData.SoundName)
}

// TestMapCrawlerAuthorWithYouTubeFields tests mapping author with Country and TotalViewCount
func TestMapCrawlerAuthorWithYouTubeFields(t *testing.T) {
	uc := implUseCase{l: &mockLogger{}}

	country := "US"
	totalViews := 5000000
	crawlerAuthor := results.CrawlerContentAuthor{
		ID:             "channel123",
		Name:           "Test Channel",
		Username:       "testchannel",
		Followers:      100000,
		Following:      0,
		Likes:          0,
		Videos:         50,
		IsVerified:     true,
		ProfileURL:     "https://youtube.com/@testchannel",
		Country:        &country,
		TotalViewCount: &totalViews,
	}

	projectAuthor := uc.mapCrawlerAuthorToProjectAuthor(crawlerAuthor)

	assert.Equal(t, "channel123", projectAuthor.ID)
	assert.Equal(t, "Test Channel", projectAuthor.Name)
	assert.NotNil(t, projectAuthor.Country)
	assert.Equal(t, "US", *projectAuthor.Country)
	assert.NotNil(t, projectAuthor.TotalViewCount)
	assert.Equal(t, 5000000, *projectAuthor.TotalViewCount)
}

// TestMapCrawlerAuthorWithoutYouTubeFields tests mapping author without YouTube fields (TikTok case)
func TestMapCrawlerAuthorWithoutYouTubeFields(t *testing.T) {
	uc := implUseCase{l: &mockLogger{}}

	crawlerAuthor := results.CrawlerContentAuthor{
		ID:             "user123",
		Name:           "Test User",
		Username:       "testuser",
		Followers:      50000,
		Following:      200,     // TikTok-specific
		Likes:          1000000, // TikTok-specific
		Videos:         100,
		IsVerified:     false,
		ProfileURL:     "https://tiktok.com/@testuser",
		Country:        nil, // TikTok doesn't have country
		TotalViewCount: nil, // TikTok doesn't have total view count
	}

	projectAuthor := uc.mapCrawlerAuthorToProjectAuthor(crawlerAuthor)

	assert.Equal(t, "user123", projectAuthor.ID)
	assert.Equal(t, "Test User", projectAuthor.Name)
	assert.Nil(t, projectAuthor.Country)          // Should be nil for TikTok
	assert.Nil(t, projectAuthor.TotalViewCount)   // Should be nil for TikTok
	assert.Equal(t, 200, projectAuthor.Following) // TikTok field preserved
	assert.Equal(t, 1000000, projectAuthor.Likes) // TikTok field preserved
}

// TestMapCrawlerCommentsWithIsFavoritedTrue tests mapping comments with is_favorited true
func TestMapCrawlerCommentsWithIsFavoritedTrue(t *testing.T) {
	ctx := context.Background()
	uc := implUseCase{l: &mockLogger{}}

	crawlerComments := []results.CrawlerComment{
		{
			ID:           "comment123",
			PostID:       "video123",
			User:         results.CrawlerCommentUser{Name: "Commenter"},
			Text:         "Great video!",
			Likes:        100,
			RepliesCount: 5,
			PublishedAt:  "2024-01-15T10:00:00Z",
			IsAuthor:     false,
			IsFavorited:  true, // YouTube favorited comment
		},
	}

	projectComments, err := uc.mapCrawlerCommentsToProjectComments(ctx, "job123", "youtube", "video123", crawlerComments)

	require.NoError(t, err)
	assert.Len(t, projectComments, 1)
	assert.Equal(t, "comment123", projectComments[0].ID)
	assert.Equal(t, "Great video!", projectComments[0].Text)
	assert.True(t, projectComments[0].IsFavorited) // Should be true
}

// TestMapCrawlerCommentsWithIsFavoritedFalse tests mapping comments with is_favorited false (TikTok case)
func TestMapCrawlerCommentsWithIsFavoritedFalse(t *testing.T) {
	ctx := context.Background()
	uc := implUseCase{l: &mockLogger{}}

	crawlerComments := []results.CrawlerComment{
		{
			ID:           "comment456",
			PostID:       "video456",
			User:         results.CrawlerCommentUser{Name: "TikTok User"},
			Text:         "Nice!",
			Likes:        50,
			RepliesCount: 2,
			PublishedAt:  "2024-01-15T11:00:00Z",
			IsAuthor:     false,
			IsFavorited:  false, // TikTok default
		},
	}

	projectComments, err := uc.mapCrawlerCommentsToProjectComments(ctx, "job456", "tiktok", "video456", crawlerComments)

	require.NoError(t, err)
	assert.Len(t, projectComments, 1)
	assert.Equal(t, "comment456", projectComments[0].ID)
	assert.Equal(t, "Nice!", projectComments[0].Text)
	assert.False(t, projectComments[0].IsFavorited) // Should be false for TikTok
}

// TestMapCrawlerCommentsMultipleWithMixedFavorited tests mapping multiple comments with mixed favorited status
func TestMapCrawlerCommentsMultipleWithMixedFavorited(t *testing.T) {
	ctx := context.Background()
	uc := implUseCase{l: &mockLogger{}}

	crawlerComments := []results.CrawlerComment{
		{
			ID:           "comment1",
			PostID:       "video123",
			User:         results.CrawlerCommentUser{Name: "User1"},
			Text:         "First comment",
			Likes:        10,
			RepliesCount: 0,
			PublishedAt:  "2024-01-15T10:00:00Z",
			IsAuthor:     false,
			IsFavorited:  true,
		},
		{
			ID:           "comment2",
			PostID:       "video123",
			User:         results.CrawlerCommentUser{Name: "User2"},
			Text:         "Second comment",
			Likes:        5,
			RepliesCount: 1,
			PublishedAt:  "2024-01-15T10:05:00Z",
			IsAuthor:     false,
			IsFavorited:  false,
		},
		{
			ID:           "comment3",
			PostID:       "video123",
			User:         results.CrawlerCommentUser{Name: "User3"},
			Text:         "Third comment",
			Likes:        20,
			RepliesCount: 3,
			PublishedAt:  "2024-01-15T10:10:00Z",
			IsAuthor:     true,
			IsFavorited:  true,
		},
	}

	projectComments, err := uc.mapCrawlerCommentsToProjectComments(ctx, "job123", "youtube", "video123", crawlerComments)

	require.NoError(t, err)
	assert.Len(t, projectComments, 3)

	// Verify each comment's favorited status
	assert.True(t, projectComments[0].IsFavorited)
	assert.False(t, projectComments[1].IsFavorited)
	assert.True(t, projectComments[2].IsFavorited)
}

// TestMapCrawlerContentDataWithMediaAndTitle tests mapping with both media and title
func TestMapCrawlerContentDataWithMediaAndTitle(t *testing.T) {
	ctx := context.Background()
	uc := implUseCase{l: &mockLogger{}}

	title := "Video with Media"
	crawlerData := results.CrawlerContentData{
		Text:     "Test",
		Duration: 180,
		Title:    &title,
		Media: &results.CrawlerContentMedia{
			Type:         "audio",
			AudioPath:    "youtube/job123/video123.mp3",
			DownloadedAt: "2024-01-15T10:30:00Z",
		},
	}

	projectData, err := uc.mapCrawlerContentDataToProjectContentData(ctx, "job123", "youtube", "video123", crawlerData)

	require.NoError(t, err)
	assert.NotNil(t, projectData.Title)
	assert.Equal(t, "Video with Media", *projectData.Title)
	assert.NotNil(t, projectData.Media)
	assert.Equal(t, "audio", projectData.Media.Type)
	assert.Equal(t, "youtube/job123/video123.mp3", projectData.Media.AudioPath)
}

// TestMapCrawlerAuthorAllFieldsPopulated tests mapping with all fields populated
func TestMapCrawlerAuthorAllFieldsPopulated(t *testing.T) {
	uc := implUseCase{l: &mockLogger{}}

	bio := "Channel bio"
	avatarURL := "https://example.com/avatar.jpg"
	country := "JP"
	totalViews := 10000000

	crawlerAuthor := results.CrawlerContentAuthor{
		ID:             "channel456",
		Name:           "Full Channel",
		Username:       "fullchannel",
		Followers:      500000,
		Following:      100,
		Likes:          2000000,
		Videos:         200,
		IsVerified:     true,
		Bio:            bio,
		AvatarURL:      &avatarURL,
		ProfileURL:     "https://youtube.com/@fullchannel",
		Country:        &country,
		TotalViewCount: &totalViews,
	}

	projectAuthor := uc.mapCrawlerAuthorToProjectAuthor(crawlerAuthor)

	// Verify all fields are mapped correctly
	assert.Equal(t, "channel456", projectAuthor.ID)
	assert.Equal(t, "Full Channel", projectAuthor.Name)
	assert.Equal(t, "fullchannel", projectAuthor.Username)
	assert.Equal(t, 500000, projectAuthor.Followers)
	assert.Equal(t, 100, projectAuthor.Following)
	assert.Equal(t, 2000000, projectAuthor.Likes)
	assert.Equal(t, 200, projectAuthor.Videos)
	assert.True(t, projectAuthor.IsVerified)
	assert.Equal(t, "Channel bio", projectAuthor.Bio)
	assert.NotNil(t, projectAuthor.AvatarURL)
	assert.Equal(t, "https://example.com/avatar.jpg", *projectAuthor.AvatarURL)
	assert.Equal(t, "https://youtube.com/@fullchannel", projectAuthor.ProfileURL)
	assert.NotNil(t, projectAuthor.Country)
	assert.Equal(t, "JP", *projectAuthor.Country)
	assert.NotNil(t, projectAuthor.TotalViewCount)
	assert.Equal(t, 10000000, *projectAuthor.TotalViewCount)
}
