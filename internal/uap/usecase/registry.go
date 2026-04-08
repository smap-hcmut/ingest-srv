package usecase

import (
	"strings"

	"ingest-srv/internal/uap"
)

type parseKey struct {
	platform string
	action   string
}

type parseFunc func(rawBytes []byte, input uap.ParseAndStoreRawBatchInput, onRecord func(uap.UAPRecord)) ([]uap.UAPRecord, error)

func normalizeParseKey(platform, action string) parseKey {
	return parseKey{
		platform: strings.ToLower(strings.TrimSpace(platform)),
		action:   strings.ToLower(strings.TrimSpace(action)),
	}
}

func (uc *implUseCase) buildParseRegistry() map[parseKey]parseFunc {
	return map[parseKey]parseFunc{
		normalizeParseKey(uap.PlatformTikTok, uap.TaskTypeFullFlow):   uc.flattenTikTokFullFlow,
		normalizeParseKey(uap.PlatformYouTube, uap.TaskTypeFullFlow):  uc.flattenYouTubeFullFlow,
		normalizeParseKey(uap.PlatformFacebook, uap.TaskTypeFullFlow): uc.flattenFacebookFullFlow,
	}
}

func (uc *implUseCase) ensureParseRegistry() {
	if uc.parsers == nil {
		uc.parsers = uc.buildParseRegistry()
	}
}

func (uc *implUseCase) resolveParser(platform, action string) (parseFunc, bool) {
	uc.ensureParseRegistry()

	parser, ok := uc.parsers[normalizeParseKey(platform, action)]
	return parser, ok
}

func (uc *implUseCase) SupportsParse(platform, action string) bool {
	_, ok := uc.resolveParser(platform, action)
	return ok
}
