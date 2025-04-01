package sheet

import (
	"google.golang.org/api/sheets/v4"
)

func (r SecurityHubSpreadSheet) DeleteAllSheets(exceptSheetTitles []string) error {
	sheetz, err := r.GetAllSheets(exceptSheetTitles)
	if err != nil {
		return err
	}

	requests := make([]*sheets.Request, 0)
	for _, sheet := range sheetz {
		requests = append(requests, &sheets.Request{DeleteSheet: &sheets.DeleteSheetRequest{
			SheetId: sheet.Properties.SheetId,
		}})
	}
	if len(requests) == 0 {
		return nil
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
