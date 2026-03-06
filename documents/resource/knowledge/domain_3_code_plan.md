# Domain 3: Chat -- Chi tiết Plan Code

**Version:** 1.0  
**Last Updated:** 2026-02-16  
**Domain:** `internal/chat` -- RAG Chat Q&A

---

## I. TỔNG QUAN

### 1. Vai trò Domain

Domain Chat là **domain giá trị nhất** của hệ thống RAG Knowledge Service, chịu trách nhiệm:

- Thực hiện RAG pipeline: nhận câu hỏi → search context → gọi LLM → sinh câu trả lời
- Quản lý multi-turn conversation (lịch sử hội thoại)
- Trích xuất citations từ search results
- Quản lý context window (đảm bảo prompt không vượt giới hạn token LLM)
- Tạo gợi ý follow-up thông minh (Smart Suggestions)

### 2. Input/Output

**Input:**

- Chat request từ 2 nguồn:
  - **HTTP API**: `POST /api/v1/chat` → Public API, JWT auth
  - **HTTP API**: `GET /api/v1/conversations/{id}` → Lấy lịch sử
  - **HTTP API**: `GET /api/v1/campaigns/{id}/suggestions` → Smart suggestions

**Output:**

- Chat response bao gồm:
  - LLM-generated answer với citations `[1],[2]...`
  - Follow-up suggestions
  - Search metadata (docs_searched, docs_used, processing_time, model_used)
- Conversation history (messages list)
- Smart suggestion queries

### 3. Kiến trúc tổng quan

```
┌─────────────────────────────────────────────────────────────────┐
│                        Callers                                   │
│  ┌──────────────────┐  ┌──────────────────┐                    │
│  │ Web UI           │  │ Mobile App       │                    │
│  │ (POST /chat)     │  │ (POST /chat)     │                    │
│  └────────┬─────────┘  └────────┬─────────┘                    │
└───────────┼──────────────────────┼──────────────────────────────┘
            └──────────┬───────────┘
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                Knowledge Service (Domain: Chat)                  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                     UseCase                               │   │
│  │  Chat(ctx, sc, input) → (ChatOutput, error)              │   │
│  │  GetConversation(ctx, sc, id) → (ConversationOutput, err)│   │
│  │  ListConversations(ctx, sc, input) → ([],error)          │   │
│  │  GetSuggestions(ctx, sc, input) → (SuggestionOutput, err)│   │
│  └────────┬──────────┬──────────┬──────────┬────────────────┘   │
│           │          │          │          │                      │
│     ┌─────▼────┐ ┌───▼────┐ ┌──▼─────┐ ┌─▼──────────┐         │
│     │ search   │ │Gemini  │ │Postgre │ │  Redis     │         │
│     │ .UseCase │ │(LLM)   │ │(convos)│ │ (cache)    │         │
│     └──────────┘ └────────┘ └────────┘ └────────────┘         │
└─────────────────────────────────────────────────────────────────┘
```

### 4. Dependencies

| Dependency       | Interface                       | Vai trò                                     |
| :--------------- | :------------------------------ | :------------------------------------------ |
| `search.UseCase` | `search.UseCase`                | Tìm documents liên quan từ Qdrant           |
| `pkg/gemini`     | `gemini.IGemini`                | Gọi LLM để sinh câu trả lời                 |
| `pkg/redis`      | `redis.IRedis`                  | Cache suggestions                           |
| `projectsrv`     | `IProject`                      | Resolve campaign → project_ids (qua search) |
| PostgreSQL       | `repository.PostgresRepository` | Lưu conversations & messages                |

---

## II. DATABASE SCHEMA

### 1. Conversations Table

```sql
CREATE TABLE knowledge.conversations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id     UUID NOT NULL,
    user_id         UUID NOT NULL,
    title           VARCHAR(500),
    status          VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',  -- ACTIVE | ARCHIVED
    message_count   INT DEFAULT 0,
    last_message_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_conversations_campaign ON knowledge.conversations(campaign_id);
CREATE INDEX idx_conversations_user ON knowledge.conversations(user_id);
CREATE INDEX idx_conversations_last_msg ON knowledge.conversations(last_message_at DESC);
```

### 2. Messages Table

```sql
CREATE TABLE knowledge.messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES knowledge.conversations(id) ON DELETE CASCADE,
    role            VARCHAR(20) NOT NULL,       -- 'user' | 'assistant'
    content         TEXT NOT NULL,
    citations       JSONB,                      -- [{id, content, relevance_score}]
    search_metadata JSONB,                      -- {total_docs_searched, docs_used, processing_time_ms, model_used}
    suggestions     JSONB,                      -- ["follow-up question 1", ...]
    filters_used    JSONB,                      -- {sentiments, aspects, platforms, ...}
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_messages_conversation ON knowledge.messages(conversation_id, created_at ASC);
```

---

## III. DIRECTORY STRUCTURE

```
internal/chat/
├── delivery/
│   └── http/
│       ├── new.go                  # Factory: Handler interface + New()
│       ├── handlers.go             # Chat, GetConversation, ListConversations, GetSuggestions
│       ├── process_request.go      # Validate + extract Scope
│       ├── presenters.go           # Request/Response DTOs
│       ├── routes.go               # Route definitions
│       └── errors.go               # Error mapping
├── repository/
│   ├── interface.go                # PostgresRepository = ConversationRepository + MessageRepository
│   ├── options.go                  # Filter/Option structs
│   └── postgre/
│       ├── new.go                  # Factory
│       ├── conversation.go         # Create, GetByID, ListByCampaign, UpdateLastMessage, Archive
│       ├── conversation_query.go   # buildListQuery, buildGetByIDQuery
│       ├── conversation_build.go   # toDomain, toDB
│       ├── message.go              # Create, ListByConversation
│       ├── message_query.go        # buildListQuery
│       └── message_build.go        # toDomain, toDB
├── usecase/
│   ├── new.go                      # implUseCase{repo, searchUC, gemini, logger, cfg}
│   ├── chat.go                     # Chat: load history → search → build prompt → call LLM → save
│   ├── conversation.go             # GetConversation, ListConversations
│   ├── suggestion.go               # GetSuggestions
│   ├── prompt.go                   # buildSystemPrompt, buildContextBlock, buildHistoryBlock, manageTokenWindow
│   └── helpers.go                  # extractCitations, generateTitle, formatContext
├── interface.go                    # UseCase interface
├── types.go                        # ChatInput, ChatOutput, Citation, ConversationOutput, etc.
└── errors.go                       # Domain errors
```

---

## IV. DOMAIN-LEVEL FILES

### A. TYPES (`types.go`)

```go
package chat

import "time"

// =====================================================
// Input Types
// =====================================================

type ChatInput struct {
	CampaignID     string       // campaign scope
	ConversationID string       // "" = new conversation, non-empty = multi-turn
	Message        string       // user's question
	Filters        ChatFilters  // optional search filters
}

type ChatFilters struct {
	Sentiments []string
	Aspects    []string
	Platforms  []string
	DateFrom   *int64
	DateTo     *int64
	RiskLevels []string
}

type GetConversationInput struct {
	ConversationID string
}

type ListConversationsInput struct {
	CampaignID string
	Limit      int
	Offset     int
}

type GetSuggestionsInput struct {
	CampaignID string
}

// =====================================================
// Output Types
// =====================================================

type ChatOutput struct {
	ConversationID string      // ID of conversation (new or existing)
	Answer         string      // LLM-generated answer
	Citations      []Citation  // Source documents referenced
	Suggestions    []string    // Follow-up question suggestions
	SearchMetadata SearchMeta  // Processing stats
}

type Citation struct {
	ID             string  // analytics_id
	Content        string  // truncated source content
	RelevanceScore float64 // 0.0 - 1.0
	Platform       string
	Sentiment      string
}

type SearchMeta struct {
	TotalDocsSearched int    // total documents considered
	DocsUsed          int    // documents included in context
	ProcessingTimeMs  int64
	ModelUsed         string // e.g. "gemini-2.0-flash"
}

type ConversationOutput struct {
	ID             string
	CampaignID     string
	UserID         string
	Title          string
	Status         string
	MessageCount   int
	Messages       []MessageOutput
	LastMessageAt  *time.Time
	CreatedAt      time.Time
}

type MessageOutput struct {
	ID             string
	Role           string      // "user" | "assistant"
	Content        string
	Citations      []Citation
	SearchMetadata *SearchMeta
	Suggestions    []string
	FiltersUsed    *ChatFilters
	CreatedAt      time.Time
}

type SuggestionOutput struct {
	Suggestions []SmartSuggestion
}

type SmartSuggestion struct {
	Query       string // suggested query text
	Category    string // "trending_negative", "sentiment_shift", "comparison", "insight"
	Description string // why this is suggested
}
```

### B. ERRORS (`errors.go`)

```go
package chat

import "errors"

var (
	ErrConversationNotFound = errors.New("chat: conversation not found")
	ErrCampaignNotFound     = errors.New("chat: campaign not found")
	ErrMessageTooShort      = errors.New("chat: message too short")
	ErrMessageTooLong       = errors.New("chat: message too long")
	ErrLLMFailed            = errors.New("chat: LLM generation failed")
	ErrSearchFailed         = errors.New("chat: search failed")
	ErrRateLimitExceeded    = errors.New("chat: rate limit exceeded")
	ErrConversationArchived = errors.New("chat: conversation is archived")
)
```

### C. INTERFACE (`interface.go`)

```go
package chat

import (
	"context"
	"knowledge-srv/internal/model"
)

//go:generate mockery --name UseCase
type UseCase interface {
	// Chat Logic
	Chat(ctx context.Context, sc model.Scope, input ChatInput) (ChatOutput, error)

	// Conversation Management
	GetConversation(ctx context.Context, sc model.Scope, input GetConversationInput) (ConversationOutput, error)
	ListConversations(ctx context.Context, sc model.Scope, input ListConversationsInput) ([]ConversationOutput, error)

	// Smart Suggestions
	GetSuggestions(ctx context.Context, sc model.Scope, input GetSuggestionsInput) (SuggestionOutput, error)
}
```

---

## V. REPOSITORY LAYER

### A. INTERFACE (`repository/interface.go`)

```go
package repository

import (
	"context"
	"knowledge-srv/internal/model"
)

//go:generate mockery --name PostgresRepository
type PostgresRepository interface {
	ConversationRepository
	MessageRepository
}

type ConversationRepository interface {
	CreateConversation(ctx context.Context, opt CreateConversationOptions) (model.Conversation, error)
	GetConversationByID(ctx context.Context, id string) (model.Conversation, error)
	ListConversations(ctx context.Context, opt ListConversationsOptions) ([]model.Conversation, error)
	UpdateConversationLastMessage(ctx context.Context, opt UpdateLastMessageOptions) error
	ArchiveConversation(ctx context.Context, id string) error
}

type MessageRepository interface {
	CreateMessage(ctx context.Context, opt CreateMessageOptions) (model.Message, error)
	ListMessages(ctx context.Context, opt ListMessagesOptions) ([]model.Message, error)
}
```

### B. OPTIONS (`repository/options.go`)

```go
package repository

import "encoding/json"

type CreateConversationOptions struct {
	CampaignID string
	UserID     string
	Title      string
}

type ListConversationsOptions struct {
	CampaignID string
	UserID     string
	Status     string // optional filter
	Limit      int
	Offset     int
}

type UpdateLastMessageOptions struct {
	ConversationID string
	MessageCount   int
}

type CreateMessageOptions struct {
	ConversationID string
	Role           string // "user" | "assistant"
	Content        string
	Citations      json.RawMessage // nullable JSON
	SearchMetadata json.RawMessage // nullable JSON
	Suggestions    json.RawMessage // nullable JSON
	FiltersUsed    json.RawMessage // nullable JSON
}

type ListMessagesOptions struct {
	ConversationID string
	Limit          int  // max messages to load (default 20)
	OrderASC       bool // true = oldest first
}
```

### C. REPOSITORY IMPLEMENTATION (`repository/postgre/conversation.go`)

```go
package postgre

import (
	"context"
	"fmt"
	"time"

	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/sqlboiler"

	"github.com/volatiletech/sqlboiler/v4/boil"
)

func (r *implRepository) CreateConversation(ctx context.Context, opt repository.CreateConversationOptions) (model.Conversation, error) {
	db := r.toDBConversation(opt)
	if err := db.Insert(ctx, r.db, boil.Infer()); err != nil {
		return model.Conversation{}, fmt.Errorf("CreateConversation: %w", err)
	}
	return r.toDomainConversation(db), nil
}

func (r *implRepository) GetConversationByID(ctx context.Context, id string) (model.Conversation, error) {
	mods := r.buildGetConversationByIDQuery(id)
	db, err := sqlboiler.Conversations(mods...).One(ctx, r.db)
	if err != nil {
		return model.Conversation{}, fmt.Errorf("GetConversationByID: %w", err)
	}
	return r.toDomainConversation(db), nil
}

func (r *implRepository) ListConversations(ctx context.Context, opt repository.ListConversationsOptions) ([]model.Conversation, error) {
	mods := r.buildListConversationsQuery(opt)
	dbConvos, err := sqlboiler.Conversations(mods...).All(ctx, r.db)
	if err != nil {
		return nil, fmt.Errorf("ListConversations: %w", err)
	}
	return r.toDomainConversationList(dbConvos), nil
}

func (r *implRepository) UpdateConversationLastMessage(ctx context.Context, opt repository.UpdateLastMessageOptions) error {
	now := time.Now()
	_, err := sqlboiler.Conversations(
		sqlboiler.ConversationWhere.ID.EQ(opt.ConversationID),
	).UpdateAll(ctx, r.db, sqlboiler.M{
		"message_count":   opt.MessageCount,
		"last_message_at": now,
		"updated_at":      now,
	})
	if err != nil {
		return fmt.Errorf("UpdateConversationLastMessage: %w", err)
	}
	return nil
}

func (r *implRepository) ArchiveConversation(ctx context.Context, id string) error {
	_, err := sqlboiler.Conversations(
		sqlboiler.ConversationWhere.ID.EQ(id),
	).UpdateAll(ctx, r.db, sqlboiler.M{
		"status":     "ARCHIVED",
		"updated_at": time.Now(),
	})
	return err
}
```

### D. MESSAGE REPOSITORY (`repository/postgre/message.go`)

```go
package postgre

import (
	"context"
	"fmt"

	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/sqlboiler"

	"github.com/volatiletech/sqlboiler/v4/boil"
)

func (r *implRepository) CreateMessage(ctx context.Context, opt repository.CreateMessageOptions) (model.Message, error) {
	db := r.toDBMessage(opt)
	if err := db.Insert(ctx, r.db, boil.Infer()); err != nil {
		return model.Message{}, fmt.Errorf("CreateMessage: %w", err)
	}
	return r.toDomainMessage(db), nil
}

func (r *implRepository) ListMessages(ctx context.Context, opt repository.ListMessagesOptions) ([]model.Message, error) {
	mods := r.buildListMessagesQuery(opt)
	dbMsgs, err := sqlboiler.Messages(mods...).All(ctx, r.db)
	if err != nil {
		return nil, fmt.Errorf("ListMessages: %w", err)
	}
	return r.toDomainMessageList(dbMsgs), nil
}
```
---

## VI. USECASE IMPLEMENTATION

### A. FACTORY (`usecase/new.go`)

```go
package usecase

import (
	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/internal/search"
	"knowledge-srv/pkg/gemini"
	"knowledge-srv/pkg/log"
)

type Config struct {
	MaxHistoryMessages int     // default 20
	MaxSearchDocs      int     // default 10
	MaxDocContentLen   int     // default 500 chars
	MinMessageLength   int     // default 3
	MaxMessageLength   int     // default 2000
	MaxTokenWindow     int     // default 28000
	GeminiModel        string  // e.g. "gemini-2.0-flash"
}

type implUseCase struct {
	repo     repository.PostgresRepository
	searchUC search.UseCase
	gemini   gemini.IGemini
	l        log.Logger
	cfg      Config
}

func New(
	repo repository.PostgresRepository,
	searchUC search.UseCase,
	gemini gemini.IGemini,
	l log.Logger,
	cfg Config,
) chat.UseCase {
	if cfg.MaxHistoryMessages <= 0 { cfg.MaxHistoryMessages = 20 }
	if cfg.MaxSearchDocs <= 0 { cfg.MaxSearchDocs = 10 }
	if cfg.MaxDocContentLen <= 0 { cfg.MaxDocContentLen = 500 }
	if cfg.MinMessageLength <= 0 { cfg.MinMessageLength = 3 }
	if cfg.MaxMessageLength <= 0 { cfg.MaxMessageLength = 2000 }
	if cfg.MaxTokenWindow <= 0 { cfg.MaxTokenWindow = 28000 }
	return &implUseCase{repo: repo, searchUC: searchUC, gemini: gemini, l: l, cfg: cfg}
}
```

### B. CHAT LOGIC (`usecase/chat.go`)

Core RAG pipeline — the most critical file.

**Flow:** validate → resolve/create conversation → load history → search → build prompt → LLM → save → return

```go
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/search"
)

func (uc *implUseCase) Chat(ctx context.Context, sc model.Scope, input chat.ChatInput) (chat.ChatOutput, error) {
	startTime := time.Now()

	// Step 0: Validate
	if err := uc.validateChatInput(input); err != nil {
		return chat.ChatOutput{}, err
	}

	// Step 1: Resolve or create conversation
	var conversation model.Conversation
	var history []model.Message
	isNew := input.ConversationID == ""

	if isNew {
		title := uc.generateTitle(input.Message)
		conv, err := uc.repo.CreateConversation(ctx, repository.CreateConversationOptions{
			CampaignID: input.CampaignID, UserID: sc.UserID, Title: title,
		})
		if err != nil {
			return chat.ChatOutput{}, fmt.Errorf("create conversation: %w", err)
		}
		conversation = conv
	} else {
		conv, err := uc.repo.GetConversationByID(ctx, input.ConversationID)
		if err != nil { return chat.ChatOutput{}, chat.ErrConversationNotFound }
		if conv.Status == "ARCHIVED" { return chat.ChatOutput{}, chat.ErrConversationArchived }
		conversation = conv

		msgs, _ := uc.repo.ListMessages(ctx, repository.ListMessagesOptions{
			ConversationID: conversation.ID, Limit: uc.cfg.MaxHistoryMessages, OrderASC: true,
		})
		history = msgs
	}

	// Step 2: Search relevant documents
	searchInput := search.SearchInput{
		CampaignID: input.CampaignID, Query: input.Message,
		Limit: uc.cfg.MaxSearchDocs, MinScore: 0.65,
		Filters: search.SearchFilters{
			Sentiments: input.Filters.Sentiments, Aspects: input.Filters.Aspects,
			Platforms: input.Filters.Platforms, DateFrom: input.Filters.DateFrom,
			DateTo: input.Filters.DateTo, RiskLevels: input.Filters.RiskLevels,
		},
	}
	searchOutput, err := uc.searchUC.Search(ctx, sc, searchInput)
	if err != nil {
		return chat.ChatOutput{}, fmt.Errorf("%w: %v", chat.ErrSearchFailed, err)
	}

	// Step 3: Build LLM prompt
	prompt := uc.buildPrompt(input.Message, searchOutput.Results, history)

	// Step 4: Call LLM
	answer, err := uc.gemini.Generate(ctx, prompt)
	if err != nil {
		return chat.ChatOutput{}, fmt.Errorf("%w: %v", chat.ErrLLMFailed, err)
	}

	// Step 5: Extract citations + suggestions
	citations := uc.extractCitations(searchOutput.Results)
	suggestions := uc.generateSuggestions(input.Message, searchOutput)

	// Step 6: Save messages (user + assistant)
	filtersJSON, _ := json.Marshal(input.Filters)
	uc.repo.CreateMessage(ctx, repository.CreateMessageOptions{
		ConversationID: conversation.ID, Role: "user",
		Content: input.Message, FiltersUsed: filtersJSON,
	})

	citationsJSON, _ := json.Marshal(citations)
	suggestionsJSON, _ := json.Marshal(suggestions)
	searchMeta := chat.SearchMeta{
		TotalDocsSearched: searchOutput.TotalFound, DocsUsed: len(citations),
		ProcessingTimeMs: time.Since(startTime).Milliseconds(), ModelUsed: uc.cfg.GeminiModel,
	}
	searchMetaJSON, _ := json.Marshal(searchMeta)
	uc.repo.CreateMessage(ctx, repository.CreateMessageOptions{
		ConversationID: conversation.ID, Role: "assistant", Content: answer,
		Citations: citationsJSON, SearchMetadata: searchMetaJSON, Suggestions: suggestionsJSON,
	})

	// Step 7: Update conversation
	uc.repo.UpdateConversationLastMessage(ctx, repository.UpdateLastMessageOptions{
		ConversationID: conversation.ID, MessageCount: conversation.MessageCount + 2,
	})

	return chat.ChatOutput{
		ConversationID: conversation.ID, Answer: answer,
		Citations: citations, Suggestions: suggestions, SearchMetadata: searchMeta,
	}, nil
}

func (uc *implUseCase) validateChatInput(input chat.ChatInput) error {
	if input.CampaignID == "" { return chat.ErrCampaignNotFound }
	if len(input.Message) < uc.cfg.MinMessageLength { return chat.ErrMessageTooShort }
	if len(input.Message) > uc.cfg.MaxMessageLength { return chat.ErrMessageTooLong }
	return nil
}
```

### C. PROMPT BUILDER (`usecase/prompt.go`)

```go
package usecase

import (
	"fmt"
	"strings"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/search"
)

const systemPrompt = `Bạn là trợ lý phân tích dữ liệu SMAP.
- Trả lời dựa trên context documents, trích dẫn bằng [1],[2]...
- Nếu không có context phù hợp, nói rõ "Không tìm thấy dữ liệu liên quan"
- Trả lời bằng tiếng Việt, ngắn gọn, chính xác`

func (uc *implUseCase) buildPrompt(question string, docs []search.SearchResult, history []model.Message) string {
	var b strings.Builder
	b.WriteString(systemPrompt + "\n\n")

	// Context block
	if len(docs) > 0 {
		b.WriteString("Context:\n")
		for i, doc := range docs {
			content := doc.Content
			if len(content) > uc.cfg.MaxDocContentLen {
				content = content[:uc.cfg.MaxDocContentLen] + "..."
			}
			b.WriteString(fmt.Sprintf("[%d] \"%s\" (Platform: %s, Sentiment: %s)\n", i+1, content, doc.Platform, doc.OverallSentiment))
		}
		b.WriteString("\n")
	}

	// History block
	if len(history) > 0 {
		b.WriteString("Conversation History:\n")
		for _, msg := range history {
			b.WriteString(fmt.Sprintf("%s: %s\n", strings.Title(msg.Role), msg.Content))
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("User: %s\nAssistant:", question))

	// Token window management (~4 chars = 1 token)
	prompt := b.String()
	if len(prompt)/4 > uc.cfg.MaxTokenWindow {
		// Reduce: fewer docs (5), fewer history (10)
		reducedDocs := docs
		if len(reducedDocs) > 5 { reducedDocs = reducedDocs[:5] }
		reducedHistory := history
		if len(reducedHistory) > 10 { reducedHistory = reducedHistory[len(reducedHistory)-10:] }
		return uc.buildPrompt(question, reducedDocs, reducedHistory)
	}
	return prompt
}
```

### D. HELPERS (`usecase/helpers.go`)

```go
package usecase

import (
	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/search"
)

func (uc *implUseCase) extractCitations(results []search.SearchResult) []chat.Citation {
	citations := make([]chat.Citation, 0, len(results))
	for _, r := range results {
		content := r.Content
		if len(content) > 200 { content = content[:200] + "..." }
		citations = append(citations, chat.Citation{
			ID: r.ID, Content: content, RelevanceScore: r.Score,
			Platform: r.Platform, Sentiment: r.OverallSentiment,
		})
	}
	return citations
}

func (uc *implUseCase) generateTitle(message string) string {
	if len(message) <= 50 { return message }
	return message[:50] + "..."
}

func (uc *implUseCase) generateSuggestions(query string, output search.SearchOutput) []string {
	suggestions := make([]string, 0, 3)
	for _, a := range output.Aggregations.ByAspect {
		if a.AvgSentimentScore < -0.3 {
			suggestions = append(suggestions, "Chi tiết về "+a.AspectDisplayName+" thì sao?")
		}
		if len(suggestions) >= 3 { break }
	}
	if len(output.Aggregations.ByPlatform) > 1 {
		suggestions = append(suggestions, "So sánh giữa các nền tảng?")
	}
	if len(suggestions) < 3 {
		suggestions = append(suggestions, "Xu hướng theo thời gian?")
	}
	if len(suggestions) > 3 { suggestions = suggestions[:3] }
	return suggestions
}
```

### E. CONVERSATION MANAGEMENT (`usecase/conversation.go`)

```go
package usecase

import (
	"context"
	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/internal/model"
)

func (uc *implUseCase) GetConversation(ctx context.Context, sc model.Scope, input chat.GetConversationInput) (chat.ConversationOutput, error) {
	conv, err := uc.repo.GetConversationByID(ctx, input.ConversationID)
	if err != nil { return chat.ConversationOutput{}, chat.ErrConversationNotFound }

	msgs, _ := uc.repo.ListMessages(ctx, repository.ListMessagesOptions{
		ConversationID: conv.ID, OrderASC: true,
	})
	return uc.toConversationOutput(conv, msgs), nil
}

func (uc *implUseCase) ListConversations(ctx context.Context, sc model.Scope, input chat.ListConversationsInput) ([]chat.ConversationOutput, error) {
	limit := input.Limit
	if limit <= 0 { limit = 20 }
	convos, err := uc.repo.ListConversations(ctx, repository.ListConversationsOptions{
		CampaignID: input.CampaignID, UserID: sc.UserID, Limit: limit, Offset: input.Offset,
	})
	if err != nil { return nil, err }

	results := make([]chat.ConversationOutput, len(convos))
	for i, c := range convos { results[i] = uc.toConversationOutput(c, nil) }
	return results, nil
}
```

---

## VII. DELIVERY LAYER

### A. ROUTES (`delivery/http/routes.go`)

```go
package http

import (
	"github.com/gin-gonic/gin"
	"knowledge-srv/internal/chat"
	"knowledge-srv/pkg/log"
	"knowledge-srv/pkg/middleware"
)

type Handler interface {
	RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware)
}

func New(l log.Logger, uc chat.UseCase) Handler {
	return &handler{l: l, uc: uc}
}

func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware) {
	api := r.Group("/api/v1")
	api.Use(mw.JWTAuth())
	{
		api.POST("/chat", h.Chat)
		api.GET("/conversations/:conversation_id", h.GetConversation)
		api.GET("/campaigns/:campaign_id/conversations", h.ListConversations)
		api.GET("/campaigns/:campaign_id/suggestions", h.GetSuggestions)
	}
}
```

### B. PRESENTERS (`delivery/http/presenters.go`)

Request/Response DTOs follow the same pattern as `domain_2_code_plan.md`. Key DTOs:

- `chatReq` → `chat.ChatInput` via `toInput()`
- `chatResp` ← `chat.ChatOutput` via `newChatResp()`
- `conversationResp` ← `chat.ConversationOutput`
- `suggestionResp` ← `chat.SuggestionOutput`

### C. ERRORS (`delivery/http/errors.go`)

Maps domain errors to HTTP errors using `pkg/errors.NewHTTPError`:

| Domain Error              | HTTP Code | Message                    |
| ------------------------- | --------- | -------------------------- |
| `ErrConversationNotFound` | 404       | "Conversation not found"   |
| `ErrCampaignNotFound`     | 404       | "Campaign not found"       |
| `ErrMessageTooShort`      | 400       | "Message too short"        |
| `ErrMessageTooLong`       | 400       | "Message too long"         |
| `ErrLLMFailed`            | 500       | "AI generation failed"     |
| `ErrConversationArchived` | 400       | "Conversation is archived" |

---

## VIII. MODEL ENTITIES

New models in `internal/model/`:

### `conversation.go`

```go
type Conversation struct {
	ID, CampaignID, UserID, Title, Status string
	MessageCount int
	LastMessageAt *time.Time
	CreatedAt, UpdatedAt time.Time
}
```

### `message.go`

```go
type Message struct {
	ID, ConversationID, Role, Content string
	Citations, SearchMetadata, Suggestions, FiltersUsed json.RawMessage
	CreatedAt time.Time
}
```

---

## IX. TESTING STRATEGY

### Unit Tests (UseCase)

| File                   | Key Test Cases                                                                                            |
| ---------------------- | --------------------------------------------------------------------------------------------------------- |
| `chat_test.go`         | NewConversation, MultiTurn, ConversationNotFound, Archived, SearchFailed, LLMFailed, MessageTooShort/Long |
| `prompt_test.go`       | WithHistory, NoHistory, NoSearchResults, TokenWindowUnder/OverLimit                                       |
| `helpers_test.go`      | ExtractCitations, GenerateTitle_Short/Long, GenerateSuggestions                                           |
| `conversation_test.go` | GetConversation_Found/NotFound, ListConversations                                                         |

### Integration Tests (Repository)

| File                   | Key Test Cases                                              |
| ---------------------- | ----------------------------------------------------------- |
| `conversation_test.go` | Create, GetByID, ListByCampaign, UpdateLastMessage, Archive |
| `message_test.go`      | Create, ListOrdered, ListLimited                            |

---

## X. WIRING (`internal/httpserver/domain_chat.go`)

```go
func (s *HTTPServer) initChatDomain() {
	repo := chatRepo.New(s.db)
	uc := chatUC.New(repo, s.searchUC, s.gemini, s.logger, chatUC.Config{
		MaxHistoryMessages: 20, MaxSearchDocs: 10, MaxDocContentLen: 500,
		MinMessageLength: 3, MaxMessageLength: 2000, MaxTokenWindow: 28000,
		GeminiModel: "gemini-2.0-flash",
	})
	handler := chatHTTP.New(s.logger, uc)
	handler.RegisterRoutes(s.router, s.mw)
}
```
