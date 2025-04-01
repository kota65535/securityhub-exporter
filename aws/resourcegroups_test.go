package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetResourceTags(t *testing.T) {
	ctx := context.Background()
	findings, err := GetFindings(ctx, []string{"ap-northeast-1"}, []string{"Security Hub"}, []Severity{HIGH})
	idSet := mapset.NewSet[string]()
	for _, f := range findings {
		for _, r := range f.Resources {
			if !arn.IsARN(*r.Id) {
				t.Logf("resource ID '%s' is not ARN, skip to get the tags.\n", *r.Id)
				continue
			}
			idSet.Add(*r.Id)
		}
	}
	mappings, err := GetResourcesTags(ctx, idSet.ToSlice())

	arnSet := mapset.NewSet[string]()

	assert.NoError(t, err)
	assert.True(t, len(mappings) > 0)

	for k := range mappings {
		arnSet.Add(k)
	}
	diffSet := idSet.Difference(arnSet)
	print(diffSet)
}

func TestChunk(t *testing.T) {
	ary := make([]int, 100)
	for i := 0; i < 100; i++ {
		ary[i] = i
	}
	result := chunk(ary, 20)
	assert.True(t, len(result) == 5)
}
