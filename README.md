# AWS SecurityHub Exporter

Exports AWS SecurityHub findings to Google Spreadsheet.

## Setup

1. Create Google Cloud project
2. Create Google Cloud Service Account
    1. API & Service -> Credentials -> Create Credentials -> Service Account
    2. Input Service Account Name
    3. Done
3. Generate Service Account Key
    1. API & Service -> Credentials -> Service Account
    2. Keys -> Add Key -> Create new key
    3. Key type: JSON
    4. Download the key
    5. Edit `config.yml` and set the path to the downloaded key
    ```yaml
    # Path to the Google Cloud credentials file (required)
    credentialsPath: securityhub-exporter-55e7f0620458.json
    ```
4. Enable Google Drive API & Sheet API
    - API & Service -> Enabled API & Services
5. Create Google Drive Folder
    - Create a folder in Google Drive where the spreadsheet will be exported
    - Edit `config.yml` and set the folder ID
    ```yaml
    # Google Drive folder ID where the spreadsheet is exported (required)
    folderId: 1U6Tz5-3qfgolLWwWICVDMPBlVzFOx7en
    ```

## Run

```
./securityhub-exporter-darwin export
```
