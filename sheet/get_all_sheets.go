package sheet

import (
	"fmt"
	"golang.org/x/exp/slices"
	"google.golang.org/api/sheets/v4"
)

func (r SecurityHubSpreadSheet) GetAllSheets(exceptSheetTitles []string) ([]*sheets.Sheet, error) {
	res, err := Retry(func() (*sheets.Spreadsheet, error) {
		return r.Service.Spreadsheets.
			Get(r.Spreadsheet.SpreadsheetId).
			Do()
	})
	if err != nil {
		return nil, err
	}

	ret := make([]*sheets.Sheet, 0)
	for _, s := range res.Sheets {
		if !slices.Contains(exceptSheetTitles, s.Properties.Title) {
			ret = append(ret, s)
		}
	}
	return ret, nil
}

func (r SecurityHubSpreadSheet) GetSheet(name string) (*sheets.Sheet, error) {
	res, err := Retry(func() (*sheets.Spreadsheet, error) {
		return r.Service.Spreadsheets.
			Get(r.Spreadsheet.SpreadsheetId).
			Do()
	})
	if err != nil {
		return nil, err
	}
	for _, s := range res.Sheets {
		if s.Properties.Title == name {
			return s, nil
		}
	}
	return nil, fmt.Errorf("sheet named '%s' not found", name)
}
