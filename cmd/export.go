package cmd

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/securityhub/types"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/kota65535/securityhub-exporter/aws"
	"github.com/kota65535/securityhub-exporter/cfg"
	"github.com/kota65535/securityhub-exporter/sheet"
	"github.com/spf13/viper"
	"log"
	"strings"

	"github.com/spf13/cobra"
)

type Project2Findings map[string][]types.AwsSecurityFinding

const noTagSheetName = "(No Tag)"

var configFile string
var config cfg.Config

func init() {
	c := &cobra.Command{
		Use:   "export [options]",
		Short: "Export AWS SecurityHub findings to Google Sheet.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run()
		},
	}
	c.PersistentFlags().StringVarP(&configFile, "config", "c", "config.yml", "config file path")

	rootCmd.AddCommand(c)
}

func run() error {
	viper.SetConfigFile(configFile)
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	cobra.CheckErr(err)

	err = viper.Unmarshal(&config)
	cobra.CheckErr(err)

	log.Println("Fetching findings...")
	findings, err := getFindingsWithTags(&config)
	if err != nil {
		return err
	}
	log.Printf("Got %d findings\n", len(findings))

	project2findings := groupFindingsByResourceTag(findings, config.GroupByTag)
	log.Printf("Got %d projects\n", len(project2findings))
	for p, f := range project2findings {
		log.Printf("  %d findings for '%s'\n", len(f), p)
	}

	log.Println("Initializing spreadsheet...")
	client, err := sheet.NewSpreadSheet(config)
	if err != nil {
		return err
	}

	log.Println("Deleting existing sheets...")
	err = client.DeleteAllSheets([]string{config.IndexSheetName})
	if err != nil {
		return err
	}

	log.Println("Updating sheets...")
	err = client.UpdateSheets(project2findings)
	if err != nil {
		return err
	}

	log.Println("Updating index sheets...")
	err = client.UpdateIndexSheet(project2findings)
	if err != nil {
		return err
	}

	log.Println("Finished! Click the link below to see the result:")
	log.Println("https://docs.google.com/spreadsheets/d/" + client.Spreadsheet.SpreadsheetId)
	return nil
}

func getFindingsWithTags(config *cfg.Config) ([]types.AwsSecurityFinding, error) {
	ctx := context.Background()

	findings, err := aws.GetFindings(ctx, config.Regions, config.ProductNames, config.Severities)
	if err != nil {
		return nil, err
	}

	resourceIds := mapset.NewSet[string]()
	for _, finding := range findings {
		for _, resource := range finding.Resources {
			if !isArn(*resource.Id) {
				log.Printf("resource ID '%s' is not ARN, skip to get the tags.\n", *resource.Id)
				continue
			}
			arn := fixArn(*resource.Id)
			if *resource.Id != arn {
				log.Printf("fixed ARN like resource ID: '%s' -> '%s'", *resource.Id, arn)
			}
			resourceIds.Add(arn)
		}
	}

	resourceId2Tags, err := aws.GetResourcesTags(ctx, resourceIds.ToSlice())

	for i := range findings {
		f := &findings[i]
		for j := range f.Resources {
			r := &f.Resources[j]
			if r.Tags == nil {
				r.Tags = make(map[string]string, 0)
			}
			for _, tag := range resourceId2Tags[*r.Id] {
				r.Tags[*tag.Key] = *tag.Value
			}
		}
	}

	return findings, nil
}

func groupFindingsByResourceTag(findings []types.AwsSecurityFinding, tag string) Project2Findings {
	result := make(map[string][]types.AwsSecurityFinding, 0)
	for _, f := range findings {
		for _, r := range f.Resources {
			if v, ok := r.Tags[tag]; ok {
				if _, ok := result[v]; !ok {
					result[v] = make([]types.AwsSecurityFinding, 0)
				}
				result[v] = append(result[v], f)
			} else {
				if _, ok := result[noTagSheetName]; !ok {
					result[noTagSheetName] = make([]types.AwsSecurityFinding, 0)
				}
				result[noTagSheetName] = append(result[noTagSheetName], f)
			}
		}
	}
	return result
}

func isArn(resourceId string) bool {
	return strings.HasPrefix(resourceId, "arn:aws:")
}

func fixArn(resourceId string) string {
	split := strings.Split(resourceId, ":")
	lastPart := split[len(split)-1]

	if !strings.HasPrefix(lastPart, "i-") {
		return resourceId
	}

	split[len(split)-1] = "instance/" + lastPart
	fixedResourceId := strings.Join(split, ":")
	return fixedResourceId
}
