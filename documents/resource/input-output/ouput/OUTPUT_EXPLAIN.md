# SMAP Enriched Output – Ý nghĩa & Cách Sử Dụng Insight

Tài liệu này mô tả **Output Enriched** của SMAP dùng để làm gì trong công việc marketing,
ai sử dụng, có thể query như thế nào, và insight đó hỗ trợ quyết định gì.

---

## 1. Output Enriched là gì?

Output Enriched là dữ liệu **sau khi hệ thống đã**:
- Chuẩn hoá từ UAP input
- Phân tích AI (Sentiment, Aspect, Entity…)
- Gắn ngữ cảnh business (project, campaign, impact)
- Sẵn sàng cho Dashboard, Alert, RAG Chat và Reporting

➡ Đây là **dữ liệu marketer trực tiếp sử dụng**, không phải dữ liệu kỹ thuật thô.

---

## 2. Các nhóm field & giá trị sử dụng

### 2.1 Project Context

**Ai dùng:** Marketing Manager, Brand Manager  
**Dùng để:**
- Lọc dashboard theo sản phẩm / chiến dịch
- So sánh trước – sau campaign
- Gom insight theo brand

**Query tiêu biểu:**
- Sentiment của VF8 trong campaign tháng 2
- So sánh VF8 với đối thủ trong cùng thời gian

---

### 2.2 Identity & Source

**Ai dùng:** Analyst, PR, Manager  
**Dùng để:**
- Truy ngược bài viết/comment gốc
- Trích dẫn bằng chứng trong báo cáo
- Xác định nền tảng gây ảnh hưởng (Facebook, TikTok…)

**Insight hỗ trợ:**
- “Insight này đến từ đâu?”
- “Ai là người nói?”

---

### 2.3 Content & Summary

**Ai dùng:** Marketer, Manager  
**Dùng để:**
- Đọc nhanh nội dung mà không cần mở link
- Tạo headline cho report
- Làm input cho RAG tóm tắt

**Insight hỗ trợ:**
- “Khách đang khen hay chê điều gì?”

---

### 2.4 NLP Insight (Sentiment, Aspect, Entity)

**Ai dùng:** Analyst, Strategy Team  

#### Sentiment
- Đo sức khoẻ thương hiệu
- Phát hiện khủng hoảng sớm

**Query:**
- Tỷ lệ sentiment NEGATIVE theo ngày
- Nguồn nào có sentiment xấu nhất

#### Aspect
- Hiểu rõ **chê/khen cái gì** (pin, giá, CSKH…)

**Query:**
- Khách chê VF8 vì lý do gì?
- Aspect nào tăng tiêu cực trong 7 ngày qua?

#### Entity
- Nhận diện sản phẩm, brand, tính năng

---

### 2.5 Business Impact & Alert

**Ai dùng:** Marketing Manager, PR Team  

**Dùng để:**
- Ưu tiên xử lý vấn đề
- Nhận cảnh báo sớm
- Không bỏ sót nội dung nguy hiểm

**Insight hỗ trợ:**
- “Nội dung nào cần xử lý trước?”
- “Có phải khủng hoảng thật không?”

---

### 2.6 RAG & Chat Insight

**Ai dùng:** Manager, Analyst, Product, Sales  

**Dùng để:**
- Hỏi đáp nhanh với dữ liệu
- Tổng hợp insight kèm bằng chứng

**Ví dụ câu hỏi:**
- Vì sao khách phàn nàn về pin VF8?
- Tuần này vấn đề nổi bật nhất là gì?

---

### 2.7 Provenance & Audit

**Ai dùng:** Tech Lead, Hội đồng đánh giá  

**Dùng để:**
- Chứng minh dữ liệu có nguồn gốc rõ ràng
- Audit pipeline AI
- Debug & reprocess

---

## 3. Output này hỗ trợ công việc marketing nào?

### Công việc hằng ngày
- Theo dõi sentiment brand
- Phát hiện vấn đề mới
- Trả lời câu hỏi nội bộ nhanh

### Công việc theo chiến dịch
- Đánh giá hiệu quả campaign
- Điều chỉnh thông điệp
- So sánh trước – sau

### Công việc xử lý khủng hoảng
- Phát hiện sớm
- Xác định nguyên nhân
- Ưu tiên nguồn nguy hiểm

### Công việc báo cáo
- Viết báo cáo tuần/tháng
- Trình bày insight cho sếp/khách hàng
- Có dẫn chứng rõ ràng

---

## 4. Tổng kết

UAP Input ghi nhận **sự thật dữ liệu**  
Enriched Output biến dữ liệu thành **hiểu biết & hành động**

Output này:
- Phục vụ Dashboard
- Phục vụ Alert
- Phục vụ RAG Chat
- Phục vụ quyết định marketing thực tế

➡ Đây là **đầu ra cốt lõi** của SMAP.
