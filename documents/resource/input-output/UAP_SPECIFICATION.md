# SMAP Universal Analytics Profile (UAP) Specification - v11.0 (Comprehensive & Verified)

**Nguyên tắc:** Ingest Service phẳng hóa dữ liệu thô sang chuẩn UAP. Tận dụng tối đa AI của TikTok và giữ lại các trường tài nguyên số (Downloads, Music, Subtitles).

---

## 1. Hệ thống Danh mục (Enums)

*   **`uap_type`**: `POST` (Gốc), `COMMENT` (Cấp 1), `REPLY` (Cấp 2+).
*   **`platform`**: `tiktok`, `facebook`, `youtube`.
*   **`account_type`**: `personal`, `creator`, `business`, `bot` (wishlist).
*   **`media_type`**: `video`, `image`, `carousel`.

---

## 2. Đặc tả Object POST (Video/Bài viết gốc)

```json
{
  "identity": {
    "uap_id": "tt_p_760990...",           // Prefix (tt/fb/yt) + type (p/c/r) + origin_id
    "origin_id": "760990...",             // Map: post.video_id
    "uap_type": "POST",                   // Enum: POST
    "platform": "tiktok",                 // Map: queue name (tiktok_tasks)
    "url": "https://www.tiktok.com/...",   // Map: post.url
    "task_id": "f0e87d9e...",              // Map: completion.task_id (Dùng để backtrack raw data)
    "project_id": "f0e87d9e"                // Map: get from ingest service
  },

  "hierarchy": {
    "parent_id": null,                    // Luôn null cho POST
    "root_id": "tt_p_760990...",          // Trỏ về chính nó
    "depth": 0                            // Luôn 0 cho POST
  },

  "content": {
    "text": "Nội dung video...",          // Map: post.description
    "hashtags": ["xuhuong", "vf8"],       // Map: post.hashtags (Mảng chuỗi)
    "tiktok_keywords": [                  // Map: detail.summary.keywords (TikTok AI tự động trích xuất cụm từ quan trọng)
      "đánh giá xe điện", "trải nghiệm VF8" 
    ],
    "is_shop_video": false,               // Map: post.is_shop_video (Phân biệt bài Review tự nhiên hay ghim link Affiliate/TikTok Shop)
    "music_title": "nhạc nền...",         // Map: detail.music_title (Để biết thương hiệu đang gắn với Trend âm thanh nào)
    "music_url": "https://v77.tiktok...", // Map: detail.downloads.music (File audio gốc để Marketer nghe lại)
    "summary_title": "Đánh giá VinFast",  // Map: detail.summary.title (Tiêu đề do TikTok AI tạo)
    "subtitle_url": "https://v16...",     // Map: detail.downloads.subtitle (Dùng để làm Transcript/RAG)
    "language": "vi",                     // Map: detail.summary.language (Nếu có)
    
    // --- WISHLIST (Hiện thực sau) ---
    "external_links": [],                 // [WISHLIST]: Regex quét text lấy link Shopee/Web
    "is_edited": false,                   // [WISHLIST]: So sánh snapshot cũ để biết có sửa bài không
    "detected_entities": []               // [WISHLIST]: NER model bóc tách Brand/Product (Analysis Service làm)
  },

  "author": {
    "id": "7520108261...",                // Map: post.author.uid
    "username": "tuyenduongxa",           // Map: post.author.username
    "nickname": "Tuyến đường xa",         // Map: post.author.nickname
    "avatar": "https://...",              // Map: post.author.avatar
    "is_verified": false,                 // Map: Mặc định false
    
    // --- WISHLIST (Hiện thực sau) ---
    "account_type": "creator",            // [WISHLIST]: Phân loại dựa trên followers/is_shop
    "follower_count": 15000,              // [WISHLIST]: Cần task cào profile riêng mới có
    "location_city": "Hanoi"              // [WISHLIST]: Bóc tách từ nickname hoặc bio
  },

  "engagement": {
    "likes": 2578,                        // Map: post.likes_count
    "comments_count": 180,                // Map: post.comments_count
    "shares": 140,                        // Map: post.shares_count
    "views": 147800,                      // Map: post.views_count
    "bookmarks": 121,                     // Map: detail.bookmarks_count
    
    // --- WISHLIST (Hiện thực sau) ---
    "reply_count": 450,                   // [WISHLIST]: Tổng số reply của toàn bộ hội thoại
    "discussion_depth": 2.5,              // [WISHLIST]: reply_count / (comments_count - reply_count)
    "velocity_score": 12.5                // [WISHLIST]: Tốc độ tăng trưởng tương tác (Like/h)
  },

  "media": [
    {
      "type": "video",                    // Map: "video" (Dựa trên platform)
      "url": "https://...",               // Play URL (Có logo)
      "download_url": "https://...",      // Map: detail.downloads.video (Video sạch No-Logo cho Marketer)
      "duration": 84,                     // Map: detail.duration (seconds)
      "thumbnail": "https://..."          // Map: detail.downloads.cover
    }
  ],

  "temporal": {
    "posted_at": "2026-02-23T03:44:32Z",  // Map: post.posted_at
    "ingested_at": "2026-03-08T04:37:50Z" // Derived: time.Now() tại Ingest
  }
}
```

---

## 3. Đặc tả Object COMMENT & REPLY

```json
{
  "identity": {
    "uap_id": "tt_c_761173...",           // Map: comments.comment_id
    "origin_id": "761173...",             // Map: comments.comment_id
    "uap_type": "COMMENT",                // Enum: COMMENT | REPLY
    "platform": "tiktok"
  },

  "hierarchy": {
    "parent_id": "tt_p_...",              // Map: origin_id của cha trực tiếp
    "root_id": "tt_p_...",                // Map: Luôn trỏ về origin_id của Post gốc
    "depth": 1                            // 1 cho Comment, 2 cho Reply
  },

  "content": {
    "text": "con này đẹp mà...",          // Map: comment.content
    "external_links": []                  // [WISHLIST]: Regex quét link trong comment
  },

  "author": {
    "id": "6830415126...",                // Map: comment.author.uid
    "username": "hanguyenhiep",           // Map: comment.author.username (Định danh duy nhất sau uid để hiển thị Dashboard)
    "nickname": "Ha Nguyen Hiep",         // Map: comment.author.nickname (Có thể bị trùng)
    "avatar": "https://..."               // Map: comment.author.avatar (Để hiển thị lên Dashboard)
  },

  "engagement": {
    "likes": 47,                          // Map: comment.likes_count
    "reply_count": 7,                     // Map: comment.reply_count (Chỉ có ở COMMENT)
    "sort_score": 0.3481                  // Map: comment.sort_extra_score.show_more_score / reply_score (Xếp hạng Bình luận Top - Phục vụ Usecase Khủng hoảng & Định hướng dư luận)
  },

  "temporal": {
    "posted_at": "2026-02-28T02:38:51Z"   // MAP: comment.commented_at (nếu là COMMENT) hoặc reply.replied_at (nếu là REPLY)
  }
}
```

---

## 4. Giải thích ý nghĩa & Mapping (For BI & Marketers)

| Nhóm trường | Trường (Field) | Tại sao Marketer cần? | Mapping thực tế (Raw JSON) |
| :--- | :--- | :--- | :--- |
| **AI Insight** | `tiktok_keywords` | Phân tích **Hot Keywords** ngay lập tức. | `detail.summary.keywords` |
| **AI Insight** | `sort_score` | Tìm **Bình luận Top** gây ảnh hưởng mạnh nhất. | `comment.sort_extra_score.show_more_score` |
| **Media Asset** | `download_url` | Tải video **No-Logo** để re-purpose/lưu trữ. | `detail.downloads.video` |
| **Media Asset** | `music_url` | Nghe và theo dõi trend âm thanh thương hiệu. | `detail.downloads.music` |
| **Media Asset** | `subtitle_url` | Cơ sở để AI tạo **Transcript** hội thoại. | `detail.downloads.subtitle` |
| **Hierarchy** | `root_id` | Đồng bộ dữ liệu cho **RAG** tìm kiếm theo ngữ cảnh. | `post.video_id` |
| **Wishlist** | `discussion_depth`| Chỉ số đo độ Viral và tính tranh luận của bài viết. | `reply_count / (comments_count - reply_count)` |

---

### Ghi chú cho Ingest Service:
1.  **Flattening**: Tách bạch POST, COMMENT, REPLY thành các record độc lập.
2.  **ID Consistency**: Đảm bảo `uap_id` là duy nhất và `root_id` luôn trỏ về Post gốc.
3.  **No Data Loss**: Marketer cực kỳ cần các trường URL (download, music, subtitle) để khai thác nội dung sạch.
