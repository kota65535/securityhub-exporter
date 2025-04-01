package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	"github.com/aws/aws-sdk-go-v2/service/securityhub/types"
)

func GetFindings(ctx context.Context, regions []string, productNames []string, severities []Severity) ([]types.AwsSecurityFinding, error) {
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	client := securityhub.NewFromConfig(awsConfig)

	input := createGetFindingsInput(regions, productNames, severities)

	ret := make([]types.AwsSecurityFinding, 0)

	for {
		res, err := client.GetFindings(ctx, &input)
		if err != nil {
			return nil, err
		}

		ret = append(ret, res.Findings...)

		// basically nil check is sufficient, but go's aws-sdk sometimes return empty string as nil
		if res.NextToken == nil || *res.NextToken == "" {
			break
		}

		input.NextToken = res.NextToken
	}

	return ret, nil
}

func createGetFindingsInput(regions []string, productNames []string, severities []Severity) securityhub.GetFindingsInput {
	// Filter by regions
	var regionFilters []types.StringFilter = nil
	if len(regions) > 0 {
		regionFilters = make([]types.StringFilter, len(regions))
		for i, r := range regions {
			regionFilters[i] = types.StringFilter{
				Comparison: types.StringFilterComparisonEquals,
				Value:      &r,
			}
		}
	}

	// Filter by product names
	var productNameFilters []types.StringFilter = nil
	if len(productNames) > 0 {
		productNameFilters = make([]types.StringFilter, len(productNames))
		for i, r := range productNames {
			productNameFilters[i] = types.StringFilter{
				Comparison: types.StringFilterComparisonEquals,
				Value:      &r,
			}
		}
	}

	// Filter by severities
	var severityFilters []types.StringFilter = nil
	if len(severities) > 0 {
		severityFilters = make([]types.StringFilter, len(severities))
		for i, s := range severities {
			str := string(s)
			severityFilters[i] = types.StringFilter{
				Comparison: types.StringFilterComparisonEquals,
				Value:      &str,
			}
		}
	}

	// Filter by record state
	recordState := "ACTIVE"
	recordStateFilters := []types.StringFilter{
		{
			Comparison: types.StringFilterComparisonEquals,
			Value:      &recordState,
		},
	}

	input := securityhub.GetFindingsInput{
		Filters: &types.AwsSecurityFindingFilters{
			Region:        regionFilters,
			ProductName:   productNameFilters,
			SeverityLabel: severityFilters,
			RecordState:   recordStateFilters,
		},
		MaxResults: 100,
	}

	return input
}
