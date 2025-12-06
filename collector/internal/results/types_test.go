package results

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnmarshalCrawlerContentDataWithYouTubeFields tests unmarshaling with YouTube-specific fields
func TestUnmarshalCrawlerContentDataWithYouTubeFields(t *testing.T) {
	jsonData := `{
		"text": "Test video",
		"duration": 120,
		"hashtags": ["test", "video"],
		"title": "Test YouTube Video Title"
	}`

	var data CrawlerContentData
	err := json.Unmarshal([]byte(jsonData), &data)

	require.NoError(t, err)
	assert.Equal(t, "Test video", data.Text)
	assert.Equal(t, 120, data.Duration)
	assert.NotNil(t, data.Title)
	assert.Equal(t, "Test YouTube Video Title", *data.Title)
}

// TestUnmarshalCrawlerContentDataWithoutYouTubeFields tests unmarshaling without YouTube-specific fields (TikTok case)
func TestUnmarshalCrawlerContentDataWithoutYouTubeFields(t *testing.T) {
	jsonData := `{
		"text": "Test TikTok video",
		"duration": 60,
		"hashtags": ["tiktok", "test"],
		"sound_name": "Original Sound"
	}`

	var data CrawlerContentData
	err := json.Unmarshal([]byte(jsonData), &data)

	require.NoError(t, err)
	assert.Equal(t, "Test TikTok video", data.Text)
	assert.Equal(t, 60, data.Duration)
	assert.Nil(t, data.Title) // Title should be nil for TikTok
	assert.Equal(t, "Original Sound", data.SoundName)
}

// TestUnmarshalCrawlerAuthorWithYouTubeFields tests unmarshaling author with YouTube-specific fields
func TestUnmarshalCrawlerAuthorWithYouTubeFields(t *testing.T) {
	jsonData := `{
		"id": "channel123",
		"name": "Test Channel",
		"username": "testchannel",
		"followers": 100000,
		"following": 0,
		"likes": 0,
		"videos": 50,
		"is_verified": true,
		"profile_url": "https://youtube.com/@testchannel",
		"country": "US",
		"total_view_count": 5000000
	}`

	var author CrawlerContentAuthor
	err := json.Unmarshal([]byte(jsonData), &author)

	require.NoError(t, err)
	assert.Equal(t, "channel123", author.ID)
	assert.Equal(t, "Test Channel", author.Name)
	assert.NotNil(t, author.Country)
	assert.Equal(t, "US", *author.Country)
	assert.NotNil(t, author.TotalViewCount)
	assert.Equal(t, 5000000, *author.TotalViewCount)
}

// TestUnmarshalCrawlerAuthorWithoutYouTubeFields tests unmarshaling author without YouTube-specific fields (TikTok case)
func TestUnmarshalCrawlerAuthorWithoutYouTubeFields(t *testing.T) {
	jsonData := `{
		"id": "user123",
		"name": "Test User",
		"username": "testuser",
		"followers": 50000,
		"following": 200,
		"likes": 1000000,
		"videos": 100,
		"is_verified": false,
		"profile_url": "https://tiktok.com/@testuser"
	}`

	var author CrawlerContentAuthor
	err := json.Unmarshal([]byte(jsonData), &author)

	require.NoError(t, err)
	assert.Equal(t, "user123", author.ID)
	assert.Equal(t, "Test User", author.Name)
	assert.Nil(t, author.Country)          // Country should be nil for TikTok
	assert.Nil(t, author.TotalViewCount)   // TotalViewCount should be nil for TikTok
	assert.Equal(t, 200, author.Following) // TikTok-specific field
	assert.Equal(t, 1000000, author.Likes) // TikTok-specific field
}

// TestUnmarshalCrawlerCommentWithYouTubeFavorited tests unmarshaling comment with is_favorited true
func TestUnmarshalCrawlerCommentWithYouTubeFavorited(t *testing.T) {
	jsonData := `{
		"id": "comment123",
		"post_id": "video123",
		"user": {
			"id": "user456",
			"name": "Commenter"
		},
		"text": "Great video!",
		"likes": 100,
		"replies_count": 5,
		"published_at": "2024-01-15T10:00:00Z",
		"is_author": false,
		"is_favorited": true
	}`

	var comment CrawlerComment
	err := json.Unmarshal([]byte(jsonData), &comment)

	require.NoError(t, err)
	assert.Equal(t, "comment123", comment.ID)
	assert.Equal(t, "Great video!", comment.Text)
	assert.True(t, comment.IsFavorited) // YouTube favorited comment
}

// TestUnmarshalCrawlerCommentWithoutFavorited tests unmarshaling comment without is_favorited (TikTok case)
func TestUnmarshalCrawlerCommentWithoutFavorited(t *testing.T) {
	jsonData := `{
		"id": "comment456",
		"post_id": "video456",
		"user": {
			"name": "TikTok User"
		},
		"text": "Nice!",
		"likes": 50,
		"replies_count": 2,
		"published_at": "2024-01-15T11:00:00Z",
		"is_author": false
	}`

	var comment CrawlerComment
	err := json.Unmarshal([]byte(jsonData), &comment)

	require.NoError(t, err)
	assert.Equal(t, "comment456", comment.ID)
	assert.Equal(t, "Nice!", comment.Text)
	assert.False(t, comment.IsFavorited) // Default false for TikTok
}

// TestUnmarshalCrawlerPayloadComplete tests unmarshaling complete payload with all fields
func TestUnmarshalCrawlerPayloadComplete(t *testing.T) {
	jsonData := `{
		"success": true,
		"payload": [
			{
				"meta": {
					"id": "video123",
					"platform": "youtube",
					"job_id": "job-abc-123",
					"crawled_at": "2024-01-15T10:30:00Z",
					"published_at": "2024-01-10T08:00:00Z",
					"permalink": "https://youtube.com/watch?v=123",
					"keyword_source": "test",
					"lang": "en",
					"region": "US",
					"pipeline_version": "v1",
					"fetch_status": "success"
				},
				"content": {
					"text": "Test video",
					"duration": 120,
					"hashtags": ["test"],
					"title": "YouTube Video Title"
				},
				"interaction": {
					"views": 1000,
					"likes": 100,
					"comments_count": 10,
					"shares": 0,
					"engagement_rate": 0.11,
					"updated_at": "2024-01-15T10:30:00Z"
				},
				"author": {
					"id": "channel123",
					"name": "Test Channel",
					"username": "testchannel",
					"followers": 10000,
					"following": 0,
					"likes": 0,
					"videos": 50,
					"is_verified": true,
					"profile_url": "https://youtube.com/@testchannel",
					"country": "US",
					"total_view_count": 5000000
				},
				"comments": [
					{
						"id": "comment123",
						"post_id": "video123",
						"user": {
							"id": "user456",
							"name": "Commenter"
						},
						"text": "Great!",
						"likes": 10,
						"replies_count": 0,
						"published_at": "2024-01-15T10:00:00Z",
						"is_author": false,
						"is_favorited": true
					}
				]
			}
		]
	}`

	var payload CrawlerPayload
	err := json.Unmarshal([]byte(jsonData), &payload)

	require.NoError(t, err)
	assert.True(t, payload.Success)
	assert.Len(t, payload.Payload, 1)

	content := payload.Payload[0]

	// Verify YouTube-specific fields are present
	assert.NotNil(t, content.Content.Title)
	assert.Equal(t, "YouTube Video Title", *content.Content.Title)

	assert.NotNil(t, content.Author.Country)
	assert.Equal(t, "US", *content.Author.Country)

	assert.NotNil(t, content.Author.TotalViewCount)
	assert.Equal(t, 5000000, *content.Author.TotalViewCount)

	assert.Len(t, content.Comments, 1)
	assert.True(t, content.Comments[0].IsFavorited)
}

// TestMarshalCrawlerContentDataPreservesYouTubeFields tests that marshaling preserves YouTube fields
func TestMarshalCrawlerContentDataPreservesYouTubeFields(t *testing.T) {
	title := "Test Title"
	data := CrawlerContentData{
		Text:  "Test",
		Title: &title,
	}

	jsonBytes, err := json.Marshal(data)
	require.NoError(t, err)

	var unmarshaled CrawlerContentData
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)

	assert.NotNil(t, unmarshaled.Title)
	assert.Equal(t, "Test Title", *unmarshaled.Title)
}

// TestMarshalCrawlerAuthorPreservesYouTubeFields tests that marshaling preserves YouTube author fields
func TestMarshalCrawlerAuthorPreservesYouTubeFields(t *testing.T) {
	country := "US"
	totalViews := 1000000
	author := CrawlerContentAuthor{
		ID:             "channel123",
		Name:           "Test",
		Username:       "test",
		Followers:      1000,
		Following:      0,
		Likes:          0,
		Videos:         10,
		IsVerified:     true,
		ProfileURL:     "https://youtube.com/@test",
		Country:        &country,
		TotalViewCount: &totalViews,
	}

	jsonBytes, err := json.Marshal(author)
	require.NoError(t, err)

	var unmarshaled CrawlerContentAuthor
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)

	assert.NotNil(t, unmarshaled.Country)
	assert.Equal(t, "US", *unmarshaled.Country)
	assert.NotNil(t, unmarshaled.TotalViewCount)
	assert.Equal(t, 1000000, *unmarshaled.TotalViewCount)
}
