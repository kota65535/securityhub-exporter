package aws

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetAllFindings(t *testing.T) {
	findings, err := GetFindings(context.Background(), []string{"ap-northeast-1"}, []string{"Security Hub"}, []Severity{HIGH})
	assert.NoError(t, err)
	assert.True(t, len(findings) > 0)
}
