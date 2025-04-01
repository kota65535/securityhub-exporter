package sheet

import (
	"context"
	"errors"
	"fmt"
	"github.com/kota65535/securityhub-exporter/aws"
	"github.com/kota65535/securityhub-exporter/cfg"
	"golang.org/x/image/colornames"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"image/color"
	"strings"
)

type SecurityHubSpreadSheet struct {
	Service        *sheets.Service
	Spreadsheet    *sheets.Spreadsheet
	Severities     []aws.Severity
	Colors         map[string]sheets.Color
	IndexSheetName string
	GroupByTag     string
}

func NewSpreadSheet(config cfg.Config) (*SecurityHubSpreadSheet, error) {
	ret := &SecurityHubSpreadSheet{}

	ret.Severities = config.Severities
	ret.Colors = make(map[string]sheets.Color, 0)
	for k, v := range config.Colors {
		c, err := toSheetsColor(v)
		if err != nil {
			return nil, err
		}
		ret.Colors[strings.ToUpper(string(k))] = *c
	}
	ret.IndexSheetName = config.IndexSheetName
	ret.GroupByTag = config.GroupByTag

	ctx := context.Background()

	driveService, err := drive.NewService(ctx, option.WithCredentialsFile(config.CredentialsPath))
	if err != nil {
		return nil, err
	}
	sheetsService, err := sheets.NewService(ctx, option.WithCredentialsFile(config.CredentialsPath))
	if err != nil {
		return nil, err
	}
	ret.Service = sheetsService

	// Search a file with the name in the folder
	files, err := Retry(func() (*drive.FileList, error) {
		return driveService.Files.List().
			Q(fmt.Sprintf("'%s' in parents", config.FolderId)).
			SupportsAllDrives(true).
			IncludeItemsFromAllDrives(true).
			Do()
	})
	if err != nil {
		return nil, err
	}
	fileId := ""
	for _, f := range files.Files {
		if f.Name == config.Title {
			fileId = f.Id
			break
		}
	}
	// Get the spreadsheet if exists
	if fileId != "" {
		spreadsheet, err := Retry(func() (*sheets.Spreadsheet, error) {
			return sheetsService.Spreadsheets.
				Get(fileId).
				Do()
		})
		if err != nil {
			return nil, err
		}
		ret.Spreadsheet = spreadsheet
		return ret, nil
	}

	// Create a new one if not found
	s := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Title: config.Title,
		},
	}
	spreadsheet, err := sheetsService.Spreadsheets.
		Create(s).
		Do()
	if err != nil {
		return nil, err
	}
	ret.Spreadsheet = spreadsheet

	// Initialize index sheet
	requests := []*sheets.Request{
		getIndexSheetInitializationRequest(config),
		getIndexSheetColumnCreationRequest(config),
	}
	_, err = Retry(func() (*sheets.BatchUpdateSpreadsheetResponse, error) {
		return sheetsService.Spreadsheets.
			BatchUpdate(spreadsheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{Requests: requests}).
			Do()
	})
	if err != nil {
		return nil, err
	}

	// Move into the folder
	_, err = Retry(func() (*drive.File, error) {
		return driveService.Files.
			Update(spreadsheet.SpreadsheetId, &drive.File{}).
			AddParents(config.FolderId).
			SupportsAllDrives(true).
			Do()
	})
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func getIndexSheetInitializationRequest(config cfg.Config) *sheets.Request {
	return &sheets.Request{
		UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
			Properties: &sheets.SheetProperties{
				SheetId: 0,
				Title:   config.IndexSheetName,
				GridProperties: &sheets.GridProperties{
					FrozenRowCount: 1,
				},
			},
			Fields: "title,gridProperties.frozenRowCount",
		},
	}
}

func getIndexSheetColumnCreationRequest(config cfg.Config) *sheets.Request {
	columns := []string{"Project", "Findings"}
	for _, s := range config.Severities {
		columns = append(columns, (string)(s))
	}

	values := make([]*sheets.CellData, 0)
	for i := range columns {
		values = append(values, &sheets.CellData{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &columns[i],
			},
			UserEnteredFormat: &sheets.CellFormat{
				TextFormat: &sheets.TextFormat{
					Bold: true,
				},
			},
		})
	}
	return &sheets.Request{
		UpdateCells: &sheets.UpdateCellsRequest{
			Rows: []*sheets.RowData{
				{
					Values: values,
				},
			},
			Range: &sheets.GridRange{
				SheetId:          0,
				StartRowIndex:    0,
				EndRowIndex:      1,
				StartColumnIndex: 0,
				EndColumnIndex:   0,
			},
			Fields: "userEnteredValue,UserEnteredFormat",
		},
	}
}

func toSheetsColor(colorName string) (*sheets.Color, error) {
	c, ok := colornames.Map[colorName]
	if !ok {
		var err error
		c, err = parseHexColor(colorName)
		if err != nil {
			return nil, err
		}
	}
	return &sheets.Color{
		Alpha: float64(c.A) / 255,
		Blue:  float64(c.B) / 255,
		Green: float64(c.G) / 255,
		Red:   float64(c.R) / 255,
	}, nil
}

var errInvalidFormat = errors.New("invalid format")

func parseHexColor(s string) (c color.RGBA, err error) {
	c.A = 0xff

	if s[0] != '#' {
		return c, errInvalidFormat
	}

	hexToByte := func(b byte) byte {
		switch {
		case b >= '0' && b <= '9':
			return b - '0'
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10
		}
		err = errInvalidFormat
		return 0
	}

	switch len(s) {
	case 7:
		c.R = hexToByte(s[1])<<4 + hexToByte(s[2])
		c.G = hexToByte(s[3])<<4 + hexToByte(s[4])
		c.B = hexToByte(s[5])<<4 + hexToByte(s[6])
	case 4:
		c.R = hexToByte(s[1]) * 17
		c.G = hexToByte(s[2]) * 17
		c.B = hexToByte(s[3]) * 17
	default:
		err = errInvalidFormat
	}
	return c, err
}
