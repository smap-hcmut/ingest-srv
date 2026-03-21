package project

import "strings"

func (uc implUseCase) buildEndpoint(path string) string {
	base := uc.baseURL
	if strings.HasSuffix(base, "/api/v1") {
		return base + "/internal" + path
	}
	return base + internalPrefix + path
}
