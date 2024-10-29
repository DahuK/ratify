package alibabacloud

import (
	"errors"
	"regexp"
	"strings"
)

const (
	acrNameSuffix = ".aliyuncs.com"
)

var errUnknownDomain = errors.New("invalid alibabacloud acr image format")
var domainPattern = regexp.MustCompile(
	`^(?:(?P<instanceName>[^.\s]+)-)?registry(?:-intl)?(?:-vpc)?(?:-internal)?(?:\.distributed)?\.(?P<region>[^.]+\-[^.]+)\.(?:cr\.)?aliyuncs\.com`)

type AcrMetaInfo struct {
	InstanceName string
	Region       string
}

func getRegionFromArtifict(artifact string) (*AcrMetaInfo, error) {
	if !strings.HasSuffix(artifact, acrNameSuffix) {
		return nil, errUnknownDomain
	}
	subItems := domainPattern.FindStringSubmatch(artifact)
	if len(subItems) != 3 {
		return nil, errUnknownDomain
	}
	acrMetaInfo := &AcrMetaInfo{
		Region: "",
	}
	acrMetaInfo.InstanceName = subItems[1]
	acrMetaInfo.Region = subItems[2]
	return acrMetaInfo, nil
}
