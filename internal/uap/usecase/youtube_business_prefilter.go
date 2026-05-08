package usecase

import (
	"ingest-srv/internal/uap"
	"strings"
)

const (
	youtubeBusinessPrefilterMinScore       = 0.45
	youtubeBusinessPrefilterMinDirectScore = 0.28
	youtubeBusinessPrefilterMinCrisisScore = 0.22
	youtubeBusinessMinParentOnlyLength     = 20
)

var (
	youtubeBusinessBrandTerms = []string{
		"ahamove",
		"aha move",
		"aha ship",
		"ahatruck",
		"aha truck",
		"ahamart",
		"aha mart",
		"tai xe aha",
		"tài xế aha",
	}
	youtubeBusinessLogisticsTerms = []string{
		"giao hàng",
		"giao hang",
		"vận chuyển",
		"van chuyen",
		"shipper",
		"tài xế",
		"tai xe",
		"đơn hàng",
		"don hang",
		"cod",
		"thu hộ",
		"thu ho",
		"phí giao",
		"phi giao",
		"cước",
		"cuoc",
		"xe tải",
		"xe tai",
		"xe van",
		"siêu tốc",
		"sieu toc",
		"hủy đơn",
		"huy don",
		"khách bom",
		"khach bom",
		"bom hàng",
		"bom hang",
		"ứng tiền",
		"ung tien",
		"tiền ứng",
		"tien ung",
		"hoàn ứng",
		"hoan ung",
		"rút tiền",
		"rut tien",
		"ví tài xế",
		"vi tai xe",
		"nhận đơn",
		"nhan don",
		"tự động nhận đơn",
		"tu dong nhan don",
		"nổ cuốc",
		"no cuoc",
		"cuốc xe",
		"cuoc xe",
		"chuyến xe",
		"chuyen xe",
		"bị khóa",
		"bi khoa",
		"khóa tài khoản",
		"khoa tai khoan",
		"khóa app",
		"khoa app",
		"khóa ví",
		"khoa vi",
		"bom hàng",
		"bom hang",
		"mất hàng",
		"mat hang",
		"giao chậm",
		"giao cham",
		"giao trễ",
		"giao tre",
		"hoàn tiền",
		"hoan tien",
		"đền bù",
		"den bu",
		"tổng đài",
		"tong dai",
		"hỗ trợ",
		"ho tro",
		"ứng dụng",
		"ung dung",
		"đăng ký tài xế",
		"dang ky tai xe",
	}
	youtubeBusinessCompetitorTerms = []string{
		"lalamove",
		"grab",
		"grabexpress",
		"be delivery",
		"shopee express",
		"shopeefood",
		"shopee food",
		"grabfood",
		"grab food",
		"grab bike",
		"grabbike",
		"bike plus",
		"befood",
		"be food",
		"ghn",
		"ghtk",
		"giao hàng nhanh",
		"giao hang nhanh",
		"giao hàng tiết kiệm",
		"giao hang tiet kiem",
		"viettel post",
		"vnpost",
	}
	youtubeBusinessCrisisTerms = []string{
		"lừa đảo",
		"lua dao",
		"phốt",
		"phot",
		"bóc phốt",
		"boc phot",
		"tẩy chay",
		"tay chay",
		"ăn chặn",
		"an chan",
		"bùng tiền",
		"bung tien",
		"không nhận được tiền",
		"khong nhan duoc tien",
		"không giao",
		"khong giao",
		"không hỗ trợ",
		"khong ho tro",
		"tai nạn",
		"tai nan",
		"đình công",
		"dinh cong",
		"bị khóa",
		"bi khoa",
		"khóa tài khoản",
		"khoa tai khoan",
	}
	youtubeBusinessGenericShortTerms = []string{
		"great job",
		"so nice",
		"very nice",
		"beautiful",
		"hay quá",
		"hay qua",
		"xin giá",
		"xin gia",
		"bao nhiêu",
		"bao nhieu",
		"mua ở đâu",
		"mua o dau",
	}
	youtubeBusinessOfftopicMarkers = []string{
		"nainital",
		"uttarakhand",
		"uttrakhand",
		"himachal",
		"kahani",
		"bhai",
		"samurai",
		"thưởng thức video và nhạc",
		"tải nội dung do bạn sáng tạo",
		"enjoy the videos and music you love",
		"upload original content",
	}
)

type youtubeBusinessPrefilterDecision struct {
	Keep    bool
	Score   float64
	Reasons []string
}

func (uc *implUseCase) shouldApplyYouTubeBusinessPrefilter(input uap.ParseAndStoreRawBatchInput, crawlKeyword string) bool {
	domainTypeCode := normalizeYouTubeBusinessText(input.DomainTypeCode)
	if strings.Contains(domainTypeCode, "ahamove") || strings.Contains(domainTypeCode, "logistics") {
		return true
	}

	keyword := normalizeYouTubeBusinessText(crawlKeyword)
	return containsAnyYouTubeBusinessTerm(keyword, youtubeBusinessBrandTerms)
}

func (uc *implUseCase) evaluateYouTubeBusinessPrefilter(record uap.UAPRecord, parent uap.UAPRecord) youtubeBusinessPrefilterDecision {
	contentText := normalizeYouTubeBusinessText(strings.Join([]string{
		record.Content.Title,
		record.Content.Text,
		record.Content.Subtitle,
		strings.Join(record.Content.Keywords, " "),
	}, " "))
	contextText := normalizeYouTubeBusinessText(buildYouTubeBusinessContext(record, parent))
	combinedText := strings.TrimSpace(contentText + " " + contextText)
	if combinedText == "" {
		return youtubeBusinessPrefilterDecision{Keep: false, Score: 0, Reasons: []string{"empty_text"}}
	}

	reasons := make([]string, 0, 6)
	score := 0.0
	brandDirect := containsAnyYouTubeBusinessTerm(contentText, youtubeBusinessBrandTerms)
	brandContext := containsAnyYouTubeBusinessTerm(contextText, youtubeBusinessBrandTerms)
	logisticsDirect := containsAnyYouTubeBusinessTerm(contentText, youtubeBusinessLogisticsTerms)
	logisticsContext := containsAnyYouTubeBusinessTerm(contextText, youtubeBusinessLogisticsTerms)
	competitorDirect := containsAnyYouTubeBusinessTerm(contentText, youtubeBusinessCompetitorTerms)
	competitorContext := containsAnyYouTubeBusinessTerm(contextText, youtubeBusinessCompetitorTerms)
	crisisDirect := containsAnyYouTubeBusinessTerm(contentText, youtubeBusinessCrisisTerms)
	crisisContext := containsAnyYouTubeBusinessTerm(contextText, youtubeBusinessCrisisTerms)

	if brandDirect {
		score += 0.50
		reasons = append(reasons, "brand_mentioned")
	} else if brandContext {
		score += 0.22
		reasons = append(reasons, "brand_in_context")
	}

	if logisticsDirect {
		score += 0.28
		reasons = append(reasons, "logistics_signal")
	} else if logisticsContext {
		score += 0.10
		reasons = append(reasons, "logistics_in_context")
	}

	if competitorDirect && (logisticsDirect || logisticsContext || brandDirect || brandContext) {
		score += 0.16
		reasons = append(reasons, "competitor_logistics_comparison")
	} else if competitorContext && brandContext {
		score += 0.06
		reasons = append(reasons, "competitor_in_context")
	}

	if crisisDirect && (brandDirect || brandContext || logisticsDirect || logisticsContext || competitorDirect) {
		score += 0.18
		reasons = append(reasons, "crisis_business_signal")
	} else if crisisContext && (brandDirect || logisticsDirect) {
		score += 0.08
		reasons = append(reasons, "crisis_in_context")
	}

	directBusinessSignal := brandDirect || logisticsDirect || competitorDirect
	contextBusinessSignal := brandContext || logisticsContext || competitorContext
	if len([]rune(contentText)) >= 80 && directBusinessSignal {
		score += 0.06
		reasons = append(reasons, "substantive_direct_text")
	}

	isComment := record.Identity.UAPType == uap.UAPTypeComment
	if isComment && !directBusinessSignal && !crisisDirect {
		if contextBusinessSignal && len([]rune(contentText)) >= youtubeBusinessMinParentOnlyLength {
			score = minYouTubeBusinessScore(score, 0.34)
			reasons = append(reasons, "comment_parent_context_only")
		} else {
			score = minYouTubeBusinessScore(score, 0.18)
			reasons = append(reasons, "comment_without_business_signal")
		}
	}

	if containsAnyYouTubeBusinessTerm(contentText, youtubeBusinessGenericShortTerms) && !directBusinessSignal && !crisisDirect {
		score = minYouTubeBusinessScore(score, 0.12)
		reasons = append(reasons, "generic_short_comment")
	}

	if containsAnyYouTubeBusinessTerm(contentText, youtubeBusinessOfftopicMarkers) && !directBusinessSignal && !crisisDirect {
		score = minYouTubeBusinessScore(score, 0.08)
		reasons = append(reasons, "offtopic_marker")
	}

	score = clampYouTubeBusinessScore(score)
	keep := score >= youtubeBusinessPrefilterMinScore ||
		(directBusinessSignal && score >= youtubeBusinessPrefilterMinDirectScore) ||
		(crisisDirect && (directBusinessSignal || contextBusinessSignal) && score >= youtubeBusinessPrefilterMinCrisisScore)
	if isComment && !directBusinessSignal && !crisisDirect && score < youtubeBusinessPrefilterMinScore {
		keep = false
	}

	return youtubeBusinessPrefilterDecision{
		Keep:    keep,
		Score:   score,
		Reasons: dedupeYouTubeBusinessReasons(reasons),
	}
}

func (uc *implUseCase) annotateYouTubeBusinessPrefilter(record *uap.UAPRecord, decision youtubeBusinessPrefilterDecision) {
	if record == nil {
		return
	}
	if record.PlatformMeta == nil {
		record.PlatformMeta = map[string]interface{}{}
	}

	youtubeMeta, _ := record.PlatformMeta["youtube"].(map[string]interface{})
	if youtubeMeta == nil {
		youtubeMeta = map[string]interface{}{}
	}
	youtubeMeta["business_prefilter_score"] = decision.Score
	youtubeMeta["business_prefilter_reasons"] = decision.Reasons
	record.PlatformMeta["youtube"] = youtubeMeta
}

func (uc *implUseCase) isYouTubeBusinessBoilerplate(record uap.UAPRecord) bool {
	contentText := normalizeYouTubeBusinessText(strings.Join([]string{
		record.Content.Title,
		record.Content.Text,
		record.Content.Subtitle,
		strings.Join(record.Content.Keywords, " "),
	}, " "))
	if contentText == "" {
		return false
	}
	return containsAnyYouTubeBusinessTerm(contentText, youtubeBusinessOfftopicMarkers) &&
		!containsAnyYouTubeBusinessTerm(contentText, youtubeBusinessBrandTerms) &&
		!containsAnyYouTubeBusinessTerm(contentText, youtubeBusinessLogisticsTerms) &&
		!containsAnyYouTubeBusinessTerm(contentText, youtubeBusinessCompetitorTerms)
}

func buildYouTubeBusinessContext(record uap.UAPRecord, parent uap.UAPRecord) string {
	parts := []string{
		record.CrawlKeyword,
		youtubeBusinessMetaString(record, "parent_title"),
		youtubeBusinessMetaString(record, "parent_channel_name"),
		youtubeBusinessMetaString(record, "parent_description_snippet"),
		youtubeBusinessMetaString(record, "parent_url"),
		youtubeBusinessMetaSlice(record, "parent_keywords"),
	}

	if strings.TrimSpace(parent.Identity.UAPID) != "" {
		parts = append(parts,
			parent.Content.Title,
			parent.Content.Text,
			parent.Content.Subtitle,
			strings.Join(parent.Content.Keywords, " "),
			parent.Author.Nickname,
			parent.Author.Username,
		)
	} else {
		parts = append(parts,
			record.Author.Nickname,
			record.Author.Username,
		)
	}

	return strings.Join(parts, " ")
}

func youtubeBusinessMetaString(record uap.UAPRecord, key string) string {
	youtubeMeta, _ := record.PlatformMeta["youtube"].(map[string]interface{})
	if youtubeMeta == nil {
		return ""
	}
	value, _ := youtubeMeta[key].(string)
	return value
}

func youtubeBusinessMetaSlice(record uap.UAPRecord, key string) string {
	youtubeMeta, _ := record.PlatformMeta["youtube"].(map[string]interface{})
	if youtubeMeta == nil {
		return ""
	}
	values, _ := youtubeMeta[key].([]string)
	if len(values) > 0 {
		return strings.Join(values, " ")
	}
	rawValues, _ := youtubeMeta[key].([]interface{})
	if len(rawValues) == 0 {
		return ""
	}
	parts := make([]string, 0, len(rawValues))
	for _, rawValue := range rawValues {
		parts = append(parts, strings.TrimSpace(toStringYouTubeBusinessValue(rawValue)))
	}
	return strings.Join(parts, " ")
}

func normalizeYouTubeBusinessText(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(value)), " ")
}

func containsAnyYouTubeBusinessTerm(text string, terms []string) bool {
	if text == "" {
		return false
	}
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func dedupeYouTubeBusinessReasons(reasons []string) []string {
	seen := make(map[string]struct{}, len(reasons))
	out := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		reason = strings.TrimSpace(reason)
		if reason == "" {
			continue
		}
		if _, ok := seen[reason]; ok {
			continue
		}
		seen[reason] = struct{}{}
		out = append(out, reason)
		if len(out) >= 6 {
			break
		}
	}
	return out
}

func clampYouTubeBusinessScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

func minYouTubeBusinessScore(score float64, limit float64) float64 {
	if score < limit {
		return score
	}
	return limit
}

func toStringYouTubeBusinessValue(value interface{}) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
