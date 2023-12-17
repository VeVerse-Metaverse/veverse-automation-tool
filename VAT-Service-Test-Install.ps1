#requires -Modules ScriptAsService
#requires -RunAsAdministrator

$BaseDir = "C:\VeVerse\veverse-automation-tool\"
$ServiceDir = "$BaseDir\Service"

$ScriptFile = "$BaseDir\Run-VAT-Test.ps1"
$OutputFile = "$ServiceDir\Run-VAT-Test.exe"

$ServiceName = "VeVerseAutomationTool_Test"
$ServiceDisplayName = "VeVerse Automation Tool (Test) Service"
$ServiceDescription = "VeVerse Automation Tool (Test Environment)"

Try {
    # Create scripts dir
    If (-not (Test-Path -Path $ServiceDir) ) {
        New-Item -Path $ServiceDir -ItemType Directory -Force
    }

    if (-not (Test-Path -Path $ScriptFile) ) {
        Throw "not found script file $ScriptFile"
    }

    # Create a service binary
    New-ScriptAsService -Path $ScriptFile -Destination $OutputFile -Name $ServiceName -DisplayName $ServiceDisplayName -Description $ServiceDescription -ErrorAction Stop

    # Install a service
    Install-ScriptAsService -Path $OutputFile -Name $ServiceName -Description $ServiceDescription -ErrorAction Stop
} Catch {
    Throw $_
}
