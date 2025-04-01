package sheet

import (
	"fmt"
	shTypes "github.com/aws/aws-sdk-go-v2/service/securityhub/types"
	"github.com/kota65535/securityhub-exporter/aws"
	"google.golang.org/api/sheets/v4"
	"log"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	base   = "https://%[1]s.console.aws.amazon.com/securityhub/home?region=%[1]s#/findings?search=Id%%3D"
	prefix = `\operator\:EQUALS\:`
)

func (r SecurityHubSpreadSheet) UpdateSheets(project2Findings map[string][]shTypes.AwsSecurityFinding) error {
	projects := make([]string, 0)
	for p := range project2Findings {
		projects = append(projects, p)
	}
	sort.Slice(projects, func(i, j int) bool {
		return strings.ToLower(projects[i]) < strings.ToLower(projects[j])
	})

	for _, project := range projects {
		log.Printf("Updating sheets for '%s'...", project)
		findings := project2Findings[project]
		sheetId, err := r.createSheet(project)
		if err != nil {
			return err
		}

		sortFindings(findings)

		err = r.colorSheet(sheetId, findings)
		if err != nil {
			return err
		}

		err = r.updateSheet(project, findings)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r SecurityHubSpreadSheet) createSheet(title string) (int64, error) {
	requests := []*sheets.Request{
		{
			AddSheet: &sheets.AddSheetRequest{
				Properties: &sheets.SheetProperties{
					Title: title,
					GridProperties: &sheets.GridProperties{
						FrozenRowCount: 1,
					},
				},
			},
		},
	}

	resp, err := Retry(func() (*sheets.BatchUpdateSpreadsheetResponse, error) {
		return r.Service.Spreadsheets.
			BatchUpdate(r.Spreadsheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{Requests: requests}).
			Do()
	})
	if err != nil {
		return 0, err
	}

	createdSheetID := resp.Replies[0].AddSheet.Properties.SheetId

	return createdSheetID, nil
}

func sortFindings(findings []shTypes.AwsSecurityFinding) {
	sort.Slice(findings, func(i, j int) bool {
		// Descending order of severity
		if findings[i].Severity.Label != findings[j].Severity.Label {
			return findings[i].Severity.Normalized > findings[j].Severity.Normalized
		}
		// Descending order of updated time
		t1, err1 := time.Parse(time.RFC3339, *findings[i].UpdatedAt)
		t2, err2 := time.Parse(time.RFC3339, *findings[j].UpdatedAt)
		if err1 != nil || err2 != nil {
			return true
		}
		return t1.After(t2)
	})
}

func (r SecurityHubSpreadSheet) colorSheet(sheetId int64, findings []shTypes.AwsSecurityFinding) error {
	ranges := make(map[aws.Severity][]int)
	for _, s := range aws.OrderedSeverities {
		start := -1
		end := -1
		for i, f := range findings {
			if string(f.Severity.Label) == string(s) && start == -1 {
				start = i
			}
			if string(f.Severity.Label) != string(s) && start >= 0 {
				end = i
				break
			}
		}
		if end == -1 {
			end = len(findings)
		}
		if start >= 0 && end >= 0 {
			ranges[s] = []int{start, end}
		}
	}

	requests := make([]*sheets.Request, 0)

	for s, rng := range ranges {
		color := r.Colors[string(s)]
		request := &sheets.Request{
			RepeatCell: &sheets.RepeatCellRequest{
				Cell: &sheets.CellData{
					UserEnteredFormat: &sheets.CellFormat{
						BackgroundColor: &color,
					},
				},
				Range: &sheets.GridRange{
					SheetId:          sheetId,
					StartRowIndex:    int64(rng[0] + 1),
					EndRowIndex:      int64(rng[1] + 1),
					StartColumnIndex: 0,
					EndColumnIndex:   int64(len(columnNames)),
				},
				Fields: "*",
			},
		}
		requests = append(requests, request)
	}

	_, err := Retry(func() (*sheets.BatchUpdateSpreadsheetResponse, error) {
		return r.Service.Spreadsheets.
			BatchUpdate(r.Spreadsheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{Requests: requests}).
			Do()
	})
	if err != nil {
		return err
	}

	return nil
}

func (r SecurityHubSpreadSheet) updateSheet(project string, findings []shTypes.AwsSecurityFinding) error {
	// Update sheet values
	values := createRowValues(findings)
	writeRange := project + "!A1"
	valueRange := &sheets.ValueRange{
		Values: values,
	}
	_, err := Retry(func() (*sheets.UpdateValuesResponse, error) {
		return r.Service.Spreadsheets.Values.
			Update(r.Spreadsheet.SpreadsheetId, writeRange, valueRange).
			ValueInputOption("RAW").
			Do()
	})
	if err != nil {
		return err
	}

	s, err := r.GetSheet(project)
	if err != nil {
		return err
	}
	sheetId := s.Properties.SheetId

	// Create the link for each finding
	rows := make([]*sheets.RowData, 0)
	for _, f := range findings {
		uri := createUriToFinding(*f.Id, *f.Region)
		rows = append(rows, &sheets.RowData{
			Values: []*sheets.CellData{
				{
					UserEnteredFormat: &sheets.CellFormat{
						TextFormat: &sheets.TextFormat{
							Link: &sheets.Link{
								Uri: uri,
							},
						},
					},
				},
			},
		})
	}
	requests := []*sheets.Request{
		{
			RepeatCell: &sheets.RepeatCellRequest{
				Cell: &sheets.CellData{
					UserEnteredFormat: &sheets.CellFormat{
						TextFormat: &sheets.TextFormat{
							Bold: true,
						},
					},
				},
				Range: &sheets.GridRange{
					SheetId:          sheetId,
					StartRowIndex:    0,
					EndRowIndex:      1,
					StartColumnIndex: 0,
					EndColumnIndex:   int64(len(columnNames)),
				},
				Fields: "userEnteredFormat.textFormat.bold",
			},
		},
		{
			UpdateCells: &sheets.UpdateCellsRequest{
				Rows: rows,
				Range: &sheets.GridRange{
					SheetId:          sheetId,
					StartRowIndex:    1,
					EndRowIndex:      int64(len(findings) + 1),
					StartColumnIndex: 0,
					EndColumnIndex:   1,
				},
				Fields: "userEnteredFormat.textFormat.link",
			},
		},
		{
			AutoResizeDimensions: &sheets.AutoResizeDimensionsRequest{
				Dimensions: &sheets.DimensionRange{
					SheetId:    s.Properties.SheetId,
					Dimension:  "COLUMNS",
					StartIndex: 2,
					EndIndex:   4,
				},
			},
		},
	}
	_, err = Retry(func() (*sheets.BatchUpdateSpreadsheetResponse, error) {
		return r.Service.Spreadsheets.
			BatchUpdate(r.Spreadsheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{Requests: requests}).
			Do()
	})
	if err != nil {
		return err
	}

	return nil
}

var columnNames = []interface{}{
	"ID",
	"Severity",
	"Title",
	"Resource",
	"Workflow Status",
	"Product Name",
	"Region",
	"Account ID",
	"Created at",
	"Updated at",
}

func createRowValues(findings []shTypes.AwsSecurityFinding) (values [][]interface{}) {
	values = append(values, columnNames)

	for _, e := range findings {
		findingId := *e.Id
		severity := e.Severity.Label
		title := *e.Title
		resourceId := *e.Resources[0].Id
		workFlowStatus := e.Workflow.Status
		productName := e.ProductName
		region := *e.Region
		awsAccountID := *e.AwsAccountId
		createdDate := strings.Split(*e.CreatedAt, "T")[0]
		updatedDate := strings.Split(*e.UpdatedAt, "T")[0]
		createdAt := createdDate
		updatedAt := updatedDate

		values = append(values, []interface{}{
			findingId,
			severity,
			title,
			resourceId,
			workFlowStatus,
			productName,
			region,
			awsAccountID,
			createdAt,
			updatedAt,
		})
	}
	return
}

func createUriToFinding(findingID string, region string) string {
	fstEncoding := url.QueryEscape(prefix + findingID)
	sndEncoding := url.QueryEscape(fstEncoding)
	return fmt.Sprintf(base, region) + sndEncoding
}
