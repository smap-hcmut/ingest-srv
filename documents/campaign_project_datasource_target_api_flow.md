# Campaign -> Project -> Datasource -> Target API Flow

Tài liệu này liệt kê **thứ tự call API thực tế** cho flow:

1. Tạo campaign
2. Tạo project
3. Cấu hình project config / crisis config
4. Tạo datasource
5. Tạo target
6. Dry run target nếu muốn

Base path hiện tại:

- `project-srv`: `/api/v1`
- `ingest-srv`: `/api/v1`

## 1. Create Flow Chuẩn

### Bước 1: Tạo campaign

Service: `project-srv`

- `POST /api/v1/campaigns`

Body tối thiểu:

```json
{
  "name": "Q2 2026 VinFast Resilience Watch",
  "description": "Monitor social conversation risk around VinFast launches and owner sentiment.",
  "start_date": "2026-04-01T00:00:00Z",
  "end_date": "2026-06-30T23:59:59Z"
}
```

Kết quả cần giữ lại:

- `campaign.id`

### Bước 2: Nếu cần sửa campaign ngay sau khi tạo

Service: `project-srv`

- `PUT /api/v1/campaigns/{campaignId}`

Khi nào dùng:

- đổi tên
- đổi mô tả
- đổi status
- đổi ngày bắt đầu/kết thúc

### Bước 3: Tạo project dưới campaign

Service: `project-srv`

- `POST /api/v1/campaigns/{campaignId}/projects`

Body:

```json
{
  "name": "VF8 Brand Monitoring",
  "description": "Track product sentiment, brand attacks, and creator-driven spikes around VF8.",
  "brand": "VinFast",
  "entity_type": "product",
  "entity_name": "VF8"
}
```

Kết quả cần giữ lại:

- `project.id`

### Bước 4: Nếu cần sửa project identity

Service: `project-srv`

- `PUT /api/v1/projects/{projectId}`

Khi nào dùng:

- đổi `name`
- đổi `description`
- đổi `brand`
- đổi `entity_type`
- đổi `entity_name`
- đổi `status`

### Bước 5: Cấu hình crisis / project config

Service: `project-srv`

- `PUT /api/v1/projects/{projectId}/crisis-config`

Body thực tế là config detection, gồm 4 khối:

- `keywords_trigger`
- `volume_trigger`
- `sentiment_trigger`
- `influencer_trigger`

Ví dụ rút gọn:

```json
{
  "keywords_trigger": {
    "enabled": true,
    "logic": "OR",
    "groups": [
      {
        "name": "Pin issue cluster",
        "keywords": ["pin", "chai pin", "tu hao nhanh"],
        "weight": 35
      }
    ]
  },
  "volume_trigger": {
    "enabled": true,
    "metric": "MENTIONS",
    "rules": [
      {
        "level": "CRITICAL",
        "threshold_percent_growth": 150,
        "comparison_window_hours": 1,
        "baseline": "PREVIOUS_PERIOD"
      }
    ]
  }
}
```

### Bước 6: Nếu cần đọc / sửa / xoá project config

Service: `project-srv`

- `GET /api/v1/projects/{projectId}/crisis-config`
- `PUT /api/v1/projects/{projectId}/crisis-config`
- `DELETE /api/v1/projects/{projectId}/crisis-config`

## 2. Datasource Flow

### Bước 7: Tạo datasource

Service: `ingest-srv`

- `POST /api/v1/datasources`

Lưu ý:

- hiện tại datasource là **1 platform**
- flow crawl thì `source_category = CRAWL`
- `source_type` sẽ là một trong `TIKTOK`, `FACEBOOK`, `YOUTUBE`

Ví dụ:

```json
{
  "project_id": "550e8400-e29b-41d4-a716-446655440200",
  "name": "TikTok Social Crawl",
  "description": "Track TikTok search, profile, and post signals for one project.",
  "source_type": "TIKTOK",
  "source_category": "CRAWL",
  "crawl_mode": "NORMAL",
  "crawl_interval_minutes": 11
}
```

Kết quả cần giữ lại:

- `data_source.id`

### Bước 8: Nếu cần đọc / sửa / xoá datasource

Service: `ingest-srv`

- `GET /api/v1/datasources/{dataSourceId}`
- `PUT /api/v1/datasources/{dataSourceId}`
- `POST /api/v1/datasources/{dataSourceId}/archive`
- `DELETE /api/v1/datasources/{dataSourceId}`

Khi nào dùng:

- đổi `name`
- đổi `description`
- sửa `config`
- sửa `account_ref`
- sửa `mapping_rules`
- archive datasource đã tạo nhầm hoặc không còn dùng
- delete datasource chỉ sau khi datasource đã `ARCHIVED`
- datasource delete là soft delete nếu entity có `deleted_at`

## 3. Target Flow

### Bước 9: Tạo grouped target

Service: `ingest-srv`

- `POST /api/v1/datasources/{dataSourceId}/targets/keywords`
- `POST /api/v1/datasources/{dataSourceId}/targets/profiles`
- `POST /api/v1/datasources/{dataSourceId}/targets/posts`

Lưu ý rất quan trọng:

- API hiện tại tạo **1 grouped target / 1 request**
- mỗi grouped target chứa `values[]` dùng chung 1 `crawl_interval_minutes`
- backend **không fan-out nữa** ở boundary API cho 3 zone này

#### 9.1 Keyword zone

Nếu user nhập:

```text
o to, vinfast; vf8
```

và interval chung là `4`, thì chỉ cần call:

```json
{
  "values": ["o to", "vinfast", "vf8"],
  "label": "Keyword cluster / launch watch",
  "platform_meta": {
    "inherited_source_type": "TIKTOK"
  },
  "is_active": true,
  "priority": 0,
  "crawl_interval_minutes": 4
}
```

#### 9.2 Profile zone

Nếu user gom nhiều profile URL có cùng interval thì tạo 1 grouped target:

Ví dụ:

```json
{
  "values": [
    "https://www.tiktok.com/@vinfastauto.official",
    "https://www.tiktok.com/@vinfast"
  ],
  "label": "Profile cluster / owned channels",
  "platform_meta": {
    "inherited_source_type": "TIKTOK"
  },
  "is_active": true,
  "priority": 0,
  "crawl_interval_minutes": 10
}
```

#### 9.3 Post zone

Nếu user gom nhiều post URL có cùng interval thì tạo 1 grouped target:

Ví dụ:

```json
{
  "values": [
    "https://www.tiktok.com/@vinfastauto.official/video/7470000000000000000",
    "https://www.tiktok.com/@vinfast/video/7470000000000000001"
  ],
  "label": "Post cluster / critical watchlist",
  "platform_meta": {
    "inherited_source_type": "TIKTOK"
  },
  "is_active": true,
  "priority": 0,
  "crawl_interval_minutes": 6
}
```

Kết quả cần giữ lại:

- từng `target.id` của từng grouped target đã tạo

### Bước 10: Nếu cần đọc / sửa / xoá target

Service: `ingest-srv`

Cho từng target:

- `GET /api/v1/datasources/{dataSourceId}/targets/{targetId}`
- `PUT /api/v1/datasources/{dataSourceId}/targets/{targetId}`
- `DELETE /api/v1/datasources/{dataSourceId}/targets/{targetId}`

Ngoài ra có thể list:

- `GET /api/v1/datasources/{dataSourceId}/targets`

Khi nào dùng:

- sửa interval của cả target group
- bật/tắt `is_active`
- sửa `label`
- thay toàn bộ `values[]`
- xoá target tạo nhầm

## 4. Dry Run Flow

### Bước 11: Sau khi config xong target, có thể dry run hoặc không

Service: `ingest-srv`

Dry run là **optional**:

- có thể chạy ngay sau khi tạo xong target
- hoặc không chạy và đi tiếp

Với datasource `CRAWL`, dry run hiện tại là **per target-group**

- `POST /api/v1/datasources/{dataSourceId}/dryrun`

Body:

```json
{
  "target_id": "660e8400-e29b-41d4-a716-446655440001",
  "sample_limit": 10,
  "force": false
}
```

Rule hiện tại:

- `target_id` là bắt buộc cho crawl dry run
- nếu muốn dry run nhiều grouped target thì gọi nhiều lần, mỗi lần 1 `target_id`
- dry run hiện tại là async remote runtime:
  - tạo `dryrun_result` ở `RUNNING`
  - publish task qua RabbitMQ cho `scapper-srv`
  - không tạo `scheduled_job` hay `external_task`
  - completion quay về `ingest_task_completions`
  - pass sẽ finalize `WARNING`, không phải `SUCCESS`
- nếu không truyền `sample_limit` thì mặc định lấy `10` data mẫu
- nếu không muốn dry run thì **không cần call API này**

### Bước 12: Xem kết quả dry run gần nhất hoặc lịch sử

Service: `ingest-srv`

- `GET /api/v1/datasources/{dataSourceId}/dryrun/latest?target_id={targetId}`
- `GET /api/v1/datasources/{dataSourceId}/dryrun/history?target_id={targetId}&page=1&limit=10`

Nếu muốn lấy latest/history ở mức datasource chung:

- `GET /api/v1/datasources/{dataSourceId}/dryrun/latest`
- `GET /api/v1/datasources/{dataSourceId}/dryrun/history?page=1&limit=10`

## 5. Flow Gợi Ý Theo UX

### Flow A: Chuẩn, có dry run cho từng grouped target

1. `POST /api/v1/campaigns`
2. `POST /api/v1/campaigns/{campaignId}/projects`
3. `PUT /api/v1/projects/{projectId}/crisis-config`
4. `POST /api/v1/datasources`
5. 1 hoặc nhiều lần:
   - `POST /api/v1/datasources/{dataSourceId}/targets/keywords`
   - `POST /api/v1/datasources/{dataSourceId}/targets/profiles`
   - `POST /api/v1/datasources/{dataSourceId}/targets/posts`
6. Với mỗi grouped target muốn kiểm tra:
   `POST /api/v1/datasources/{dataSourceId}/dryrun`
7. Nếu cần xem lại:
   `GET /api/v1/datasources/{dataSourceId}/dryrun/latest?target_id={targetId}`
8. Chỉ activate khi tất cả grouped target đã có latest dry run và không có target nào `FAILED`

### Flow B: Chuẩn, nhưng bỏ qua dry run

1. `POST /api/v1/campaigns`
2. `POST /api/v1/campaigns/{campaignId}/projects`
3. `PUT /api/v1/projects/{projectId}/crisis-config`
4. `POST /api/v1/datasources`
5. 1 hoặc nhiều lần:
   - `POST /api/v1/datasources/{dataSourceId}/targets/keywords`
   - `POST /api/v1/datasources/{dataSourceId}/targets/profiles`
   - `POST /api/v1/datasources/{dataSourceId}/targets/posts`
6. Dừng ở đây, không call dry run

## 6. Flow Chỉnh Sửa Sau Khi Đã Tạo

### Nếu cần sửa campaign

- `PUT /api/v1/campaigns/{campaignId}`

### Nếu cần sửa project

- `PUT /api/v1/projects/{projectId}`

### Nếu cần sửa project config

- `PUT /api/v1/projects/{projectId}/crisis-config`

### Nếu cần sửa datasource

- `PUT /api/v1/datasources/{dataSourceId}`

### Nếu cần sửa một target

- `PUT /api/v1/datasources/{dataSourceId}/targets/{targetId}`

### Nếu cần dry run lại sau khi sửa target

- `POST /api/v1/datasources/{dataSourceId}/dryrun`

Body vẫn theo target cụ thể:

```json
{
  "target_id": "{targetId}",
  "sample_limit": 10,
  "force": true
}
```

## 7. Flow Xoá / Archive Sau Khi Đã Tạo

### Nếu cần xoá crisis config

- `DELETE /api/v1/projects/{projectId}/crisis-config`

### Nếu cần xoá target

- `DELETE /api/v1/datasources/{dataSourceId}/targets/{targetId}`

### Nếu cần xoá datasource

- `POST /api/v1/datasources/{dataSourceId}/archive`
- `DELETE /api/v1/datasources/{dataSourceId}`

### Nếu cần xoá project

- `DELETE /api/v1/projects/{projectId}`

### Nếu cần xoá campaign

- `DELETE /api/v1/campaigns/{campaignId}`

## 8. Dependency Chain Cần Nhớ

Thứ tự phụ thuộc ID:

- tạo `campaign` trước để có `campaignId`
- tạo `project` sau để có `projectId`
- tạo `datasource` sau để có `dataSourceId`
- tạo grouped `target` sau để có `targetId`
- dry run target-group dùng `dataSourceId + targetId`

Tóm tắt ngắn:

- `campaignId` -> để tạo `project`
- `projectId` -> để tạo `crisis-config` và `datasource`
- `dataSourceId` -> để tạo grouped `target`
- `targetId` -> để dry run grouped target đó nếu muốn
