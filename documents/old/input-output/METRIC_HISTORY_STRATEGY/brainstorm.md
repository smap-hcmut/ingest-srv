# BRAINSTORM: CHIẾN LƯỢC LƯU TRỮ METRIC HISTORY (TIME-SERIES)

## 1. Vấn đề của thiết kế hiện tại
- Hệ thống hiện tại dùng cơ chế `UPSERT` dựa trên ID bài viết.
- **Hậu quả:** Chỉ lưu được trạng thái cuối cùng (Latest State). Mất toàn bộ lịch sử biến động (History) của tương tác (Like, Share, Comment, Rating).
- **Thiếu hụt tính năng:** Không thể phân tích xu hướng tăng trưởng (Growth Rate), phát hiện đột biến (Viral Spike Detection) của từng bài viết cụ thể.

---

## 2. Mục tiêu
Xây dựng cơ chế lưu trữ song song để đạt được cả 2 mục đích:
1.  **State (Trạng thái):** Truy xuất nhanh thông tin mới nhất để hiển thị list bài/tìm kiếm (Phục vụ UI/RAG).
2.  **Series (Chuỗi):** Lưu vết toàn bộ lịch sử thay đổi để vẽ biểu đồ và phân tích xu hướng (Phục vụ Analytics).

---

## 3. Đề xuất Kiến trúc Lưu trữ (Dual-Storage Strategy)

### 3.1. Bảng `analytic_records` 
*   **Chiến lược:** `UPSERT` (Ghi đè).
*   **Mục đích:** Lưu nội dung text, thông tin tác giả, và các chỉ số *mới nhất*. Dùng để search, filter, RAG.
*   **Update Trigger:** Mỗi khi Crawl lại thấy content cũ -> Update bảng này.

### 3.2. Bảng `metric_snapshots` (History Log - Time Series)
*   **Chiến lược:** `APPEND ONLY` (Chỉ thêm mới).
*   **Mục đích:** Lưu ảnh chụp (snapshot) các chỉ số tại thời điểm crawl.
*   **Tần suất:**
    *   **Option A (Full Log):** Lưu mọi lần crawl (Tốn dung lượng nếu crawl dày).
    *   **Option B (Daily Snapshot - Recommended):** Một ngày chỉ lưu 1 bản ghi cuối cùng/trung bình.
    *   **Option C (Smart Change):** Chỉ lưu dòng mới nếu chỉ số thay đổi > X% so với lần lưu trước.

---

## 4. Thiết kế Schema Time-Series

### Schema Logic
Chúng ta không lưu lại Text (vì tốn chỗ). Chỉ lưu các con số thay đổi.

```json
// Bảng metric_snapshots
{
  "record_id": "uuid_v4",          // ID của snapshot này
  "doc_id": "fb_post_987654321",   // Foreign Key trỏ về bài gốc (không lưu lại content)
  "project_id": "proj_1",          // Partition Key (để query nhanh theo dự án)
  "captured_at": "2026-02-08T10:00:00Z", // Thời điểm ghi nhận (Time-series Axis)
  
  // Các chỉ số biến động (Map trực tiếp từ signals.engagement của UAP Input)
  "metrics": {
    "like_count": 150,      // Match với input_explain.jsonc
    "comment_count": 45,    
    "share_count": 10,
    "view_count": 5000,
    "rating": 4.5     
  },

  // (Optional) Delta - Tốc độ tăng trưởng so với lần trước
  "delta": {
    "like_growth": 50, // +50 like
    "velocity": 5.2    // Tốc độ: 5.2 like/giờ
  }
}
```

---

## 5. Phương án Triển khai Database (Chọn 1)

### Option 1: Vẫn dùng PostgreSQL (Partitioning) - **AN TOÀN NHẤT**
*   Tận dụng Postgres 16+ với tính năng Partitioning theo thời gian (`metric_snapshots_2026_02`).
*   **Ưu:** Không cần thêm công nghệ mới. Transaction an toàn.
*   **Nhược:** Cần job dọn dẹp định kỳ (Retention Policy) nếu dữ liệu quá 1-2 năm.

### Option 2: TimescaleDB (Extension của Postgres) - **TỐI ƯU NHẤT**
*   Cài thêm extension TimescaleDB. Nó biến bảng thường thành Hypertables tối ưu cho Time-series.
*   **Ưu:** Query "Last point", "Time-bucket average" cực nhanh. Tự động nén dữ liệu cũ.
*   **Nhược:** Thêm 1 dependency phải quản lý.

### Option 3: InfluxDB / Prometheus - **KHÔNG KHUYẾN NGHỊ**
*   Chuyên cho monitor server, metric đơn giản. Khó link (join) lại với dữ liệu Text trong Postgres.

---

## 6. Logic "Smart Ingest" (Cập nhật Dataplane)

Khi Worker nhận dữ liệu từ Kafka (`ingest.uap.ready`), nó sẽ thực hiện transaction 2 bước:

1.  **Bước 1 (Upsert Master):**
    ```sql
    INSERT INTO analytic_records (doc_id, content, likes, updated_at)
    VALUES ('fb_1', 'Text...', 150, NOW())
    ON CONFLICT (doc_id) DO UPDATE 
    SET likes = 150, updated_at = NOW();
    ```

2.  **Bước 2 (Append History):**
    *   Logic kiểm tra: *"Lần snapshot cuối cùng của bài `fb_1` cách đây bao lâu?"*
    *   Nếu `> 24h` HOẶC `abs(new_like - old_like) > threadhold`:
        ```sql
        INSERT INTO metric_snapshots (doc_id, captures_at, metrics)
        VALUES ('fb_1', NOW(), '{"like": 150...}');
        ```
