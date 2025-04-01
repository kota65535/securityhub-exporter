package cfg

import "github.com/kota65535/securityhub-exporter/aws"

type Config struct {
	CredentialsPath string
	FolderId        string
	Title           string
	GroupByTag      string
	Colors          map[aws.Severity]string
	Severities      []aws.Severity
	ProductNames    []string
	Regions         []string
	IndexSheetName  string
}
