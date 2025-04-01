package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
)

type ResourceID2Tags map[string][]types.Tag
type Severity string

const (
	CRITICAL      Severity = "CRITICAL"
	HIGH          Severity = "HIGH"
	MEDIUM        Severity = "MEDIUM"
	LOW           Severity = "LOW"
	INFORMATIONAL Severity = "INFORMATIONAL"
)

var OrderedSeverities = []Severity{CRITICAL, HIGH, MEDIUM, LOW, INFORMATIONAL}
