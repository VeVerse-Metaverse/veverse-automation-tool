# Build the tool
cd X:\VeVerse\veverse-automation-tool
go mod tidy
go build -o X:\VeVerse\veverse-automation-tool\bin\veverse-automation.exe

# Set up environment
$env:VAT_API2_URL='VAT_API2_URL'
$env:VAT_API_EMAIL='VAT_API_EMAIL'
$env:VAT_API_PASSWORD='VAT_API_PASSWORD*'
$env:VAT_CERT_FILE='AT_CERT_FILE'
$env:VAT_CERT_PASSWORD='VAT_CERT_PASSWORD'
$env:VAT_EDITOR_PATH='VAT_EDITOR_PATH'
$env:VAT_LAUNCHER_DIR='VAT_LAUNCHER_DIR'
$env:VAT_PLATFORMS='VAT_PLATFORMS'
$env:VAT_PROJECT_DIR='VAT_PROJECT_DIR'
$env:VAT_PROJECT_NAME='VAT_PROJECT_NAME'
$env:VAT_SIGNTOOL_PATH='VAT_SIGNTOOL_PATH'
$env:VAT_UAT_PATH='VAT_UAT_PATH'
$env:VAT_WAILS_PATH='VAT_WAILS_PATH'
$env:VAT_UE_VERSION_CODE='VAT_UE_VERSION_CODE'
$env:VAT_UE_VERSION_MARKETPLACE='5.1'
$env:VAT_UVS_PATH='VAT_UVS_PATH'
$env:VAT_JOB_TYPES='VAT_JOB_TYPES'
$env:VAT_DEPLOYMENTS='VAT_DEPLOYMENTS'

$vat = Get-Process veverse-automation -ErrorAction SilentlyContinue
If (-not ($vat)) {
    # Run the automation tool
    X:\VeVerse\veverse-automation-tool\bin\veverse-automation.exe
}

Remove-Variable vat