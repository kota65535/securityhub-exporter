package sheet

import (
	"fmt"
	shTypes "github.com/aws/aws-sdk-go-v2/service/securityhub/types"
	"google.golang.org/api/sheets/v4"
	"strconv"
)

func (r SecurityHubSpreadSheet) UpdateIndexSheet(project2Findings map[string][]shTypes.AwsSecurityFinding) error {
	sheetz, err := r.GetAllSheets(nil)
	if err != nil {
		return err
	}

	// Clear sheet
	allRange := fmt.Sprintf("%s!A2:Z", r.IndexSheetName)
	_, err = Retry(func() (*sheets.ClearValuesResponse, error) {
		return r.Service.Spreadsheets.Values.
			Clear(r.Spreadsheet.SpreadsheetId, allRange, &sheets.ClearValuesRequest{}).
			Do()
	})
	if err != nil {
		return err
	}

	// Create links for each sheet
	rows := make([]*sheets.RowData, 0)
	for _, s := range sheetz {
		if s.Properties.Title == r.IndexSheetName {
			continue
		}
		uri := fmt.Sprintf("#gid=%d", s.Properties.SheetId)
		values := []*sheets.CellData{
			{
				UserEnteredValue: &sheets.ExtendedValue{
					StringValue: &s.Properties.Title,
				},
				UserEnteredFormat: &sheets.CellFormat{
					TextFormat: &sheets.TextFormat{
						Link: &sheets.Link{
							Uri: uri,
						},
					},
				},
			},
		}
		findings := project2Findings[s.Properties.Title]
		lenStr := strconv.Itoa(len(findings))
		values = append(values, &sheets.CellData{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &lenStr,
			},
		})
		for _, severity := range r.Severities {
			count := 0
			for _, f := range findings {
				if (string)(f.Severity.Label) == (string)(severity) {
					count++
				}
			}
			countStr := strconv.Itoa(count)
			values = append(values, &sheets.CellData{
				UserEnteredValue: &sheets.ExtendedValue{
					StringValue: &countStr,
				},
			})
		}
		rows = append(rows, &sheets.RowData{Values: values})
	}

	requests := []*sheets.Request{
		{
			UpdateCells: &sheets.UpdateCellsRequest{
				Rows: rows,
				Range: &sheets.GridRange{
					SheetId:          0,
					StartRowIndex:    1,
					EndRowIndex:      int64(len(sheetz) + 1),
					StartColumnIndex: 0,
					EndColumnIndex:   0,
				},
				Fields: "userEnteredValue,userEnteredFormat",
			},
		},
		{
			AutoResizeDimensions: &sheets.AutoResizeDimensionsRequest{
				Dimensions: &sheets.DimensionRange{
					SheetId:    0,
					Dimension:  "COLUMNS",
					StartIndex: 0,
					EndIndex:   1,
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
