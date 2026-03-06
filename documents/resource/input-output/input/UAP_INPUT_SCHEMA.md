# SMAP Unified Analytics Protocol (UAP) Schema Specification (v1.0)

*(Dành cho Developer & Data Engineer)*

Tài liệu này định nghĩa cấu trúc JSON chuẩn (Schema) được sử dụng thống nhất trong toàn bộ hệ thống SMAP. Mọi dữ liệu từ bất kỳ nguồn nào (Crawl, File Upload, Webhook) đều phải được `Ingest Worker` chuyển đổi về cấu trúc này trước khi đẩy vào `Kafka` để `Analytics` xử lý.

---

## 1. Cấu trúc Tổng quan (Root Structure)

Một bản ghi UAP (UAP Record) gồm 6 khối chính:

1. **`uap_version`**: Phiên bản format đang dùng.
2. **`event_id`**: Định danh duy nhất của sự kiện xử lý này.
3. **`ingest`**: Metadata về quá trình thu thập (Ai lấy? Lấy từ đâu? Dự án nào?).
4. **`content`**: Nội dung chính cần phân tích (Text, Ảnh, Video).
5. **`signals`**: Các tín hiệu số học (Like, Share, Rating, Geo).
6. **`context`**: Ngữ cảnh bổ sung (Campaign, Keyword matched).
7. **`raw`**: Dữ liệu gốc (để debug hoặc tra cứu lại).

---

## 2. Chi tiết từng field (Schema Definition)

### 2.1. Root Fields

| Field         | Type   | Description                                              | Ví dụ           |
| :------------ | :----- | :------------------------------------------------------- | :-------------- |
| `uap_version` | String | Phiên bản của schema, dùng để backward compatibility     | `"1.0"`         |
| `event_id`    | UUID   | ID duy nhất của message này trong hệ thống Queue (Kafka) | `"b6d6b1e2..."` |

### 2.2. Block: `ingest` (Thông tin thu thập)

Chứa thông tin quản trị: Dữ liệu này thuộc về ai và đến từ đâu.

| Field                | Type    | Description                                                            |
| :------------------- | :------ | :--------------------------------------------------------------------- |
| `project_id`         | String  | ID của Project sở hữu dữ liệu này (Quan trọng để billing/phân quyền)   |
| **`entity`**         | Object  | Thông tin về thực thể đang theo dõi (Context cho AI)                   |
| `entity.entity_type` | Enum    | Loại thực thể: `product`, `campaign`, `service`, `competitor`, `topic` |
| `entity.entity_name` | String  | Tên cụ thể (VD: "VF8", "iPhone 15")                                    |
| `entity.brand`       | String  | Thương hiệu (VD: "VinFast")                                            |
| **`source`**         | Object  | Nguồn gốc dữ liệu                                                      |
| `source.source_id`   | String  | ID cấu hình source trong DB                                            |
| `source.source_type` | Enum    | `FACEBOOK`, `TIKTOK`, `YOUTUBE`, `FILE_UPLOAD`, `WEBHOOK`              |
| `source.account_ref` | Object  | Tham chiếu đến tài khoản/file gốc (Tên Fanpage hoặc Tên File)          |
| **`batch`**          | Object  | Thông tin về đợt lấy dữ liệu                                           |
| `batch.mode`         | Enum    | `SCHEDULED_CRAWL` (Lịch), `MANUAL_UPLOAD` (Tay), `WEBHOOK` (Đẩy)       |
| `batch.received_at`  | ISO8601 | Thời điểm hệ thống nhận được dữ liệu                                   |
| **`trace`**          | Object  | Dùng cho Debug/Audit                                                   |
| `trace.raw_ref`      | URI     | Đường dẫn đến file raw gốc lưu trên MinIO/S3                           |
| `trace.mapping_id`   | String  | ID của quy tắc mapping đã dùng (nếu là File Upload)                    |

### 2.3. Block: `content` (Nội dung chính - AI Input)

Đây là phần quan trọng nhất để các Worker (Sentiment/Aspect) chạy phân tích.

| Field                   | Type    | Description                                                            |
| :---------------------- | :------ | :--------------------------------------------------------------------- |
| `doc_id`                | String  | ID gốc của bài viết bên nền tảng nguồn (Post ID, Row ID)               |
| `doc_type`              | String  | Loại nội dung: `post`, `comment`, `video`, `news`, `feedback`          |
| **`parent`**            | Object  | Nếu đây là comment, nó thuộc bài post nào? (Để dựng lại cây thảo luận) |
| `url`                   | URL     | Link trực tiếp đến bài viết gốc (nếu có)                               |
| `language`              | Code    | Mã ngôn ngữ (`vi`, `en`)                                               |
| `published_at`          | ISO8601 | Thời điểm nội dung được tạo ra bởi tác giả (Created Time)              |
| **`author`**            | Object  | Người tạo ra nội dung                                                  |
| `author.author_id`      | String  | ID người dùng bên nguồn (nếu lấy được)                                 |
| `author.display_name`   | String  | Tên hiển thị                                                           |
| `text`                  | String  | **DỮ LIỆU CHÍNH:** Văn bản cần phân tích cảm xúc/khía cạnh             |
| **`attachments`**       | Array   | Danh sách ảnh/video đính kèm                                           |
| `attachments[].type`    | Enum    | `image`, `video`, `link`                                               |
| `attachments[].content` | String  | Mô tả nội dung ảnh/video (OCR text hoặc Caption) - AI sẽ đọc phần này  |

### 2.4. Block: `signals` (Tín hiệu số - Metrics)

Dùng để lọc, sắp xếp độ hot, đánh trọng số cho bài viết.

| Field            | Type   | Description                                      |
| :--------------- | :----- | :----------------------------------------------- |
| **`engagement`** | Object | Các chỉ số tương tác mạng xã hội                 |
| `like_count`     | Int    | Số lượt thích / reaction                         |
| `comment_count`  | Int    | Số lượt bình luận                                |
| `share_count`    | Int    | Số lượt chia sẻ                                  |
| `view_count`     | Int    | Số lượt xem (Video)                              |
| `rating`         | Float  | Điểm đánh giá (Sao) nếu là Review/Feedback (1-5) |
| **`geo`**        | Object | Thông tin địa lý (nếu có)                        |
| `geo.country`    | String | Mã quốc gia                                      |

### 2.5. Block: `context` (Ngữ cảnh bổ sung)

Thông tin được làm giàu (Enriched) ngay từ lúc Ingest.

| Field              | Type   | Description                                              |
| :----------------- | :----- | :------------------------------------------------------- |
| `keywords_matched` | Array  | Bài viết này match với từ khóa monitoring nào?           |
| `campaign_id`      | String | ID chiến dịch (nếu bài viết thuộc chiến dịch đang track) |

### 2.6. Block: `raw` (Dữ liệu gốc)

Dữ liệu thô chưa qua tẩy rửa, dùng để đối chiếu khi cần.

| Field             | Type   | Description                                      |
| :---------------- | :----- | :----------------------------------------------- |
| `original_fields` | Object | Key-Value cặp dữ liệu gốc từ API hoặc File Excel |

---

## 3. Ví dụ Minh họa (JSON)

### 3.1. Trường hợp Crawl Facebook (Social Media)

```json
{
  "uap_version": "1.0",
  "ingest": {
    "entity": { "entity_type": "product", "entity_name": "VF8" },
    "source": { "source_type": "FACEBOOK" }
  },
  "content": {
    "text": "Xe đi êm nhưng pin sụt nhanh...",
    "published_at": "2026-02-07T09:55:00Z"
  },
  "signals": {
    "engagement": { "like_count": 120, "comment_count": 34 }
  }
}
```

### 3.2. Trường hợp Upload Excel (Customer Feedback)

```json
{
  "uap_version": "1.0",
  "ingest": {
    "entity": { "entity_type": "product", "entity_name": "VF8" },
    "source": { "source_type": "FILE_UPLOAD" }
  },
  "content": {
    "text": "Giá cao, pin nhanh hết",
    "published_at": "2026-02-05T08:00:00Z" // Map từ cột 'Ngày gửi'
  },
  "signals": {
    "engagement": { "rating": 3 } // Map từ cột 'Rating 3 sao'
  },
  "raw": {
    "original_fields": { "feedback": "Giá cao...", "customer": "KH A" }
  }
}
```
