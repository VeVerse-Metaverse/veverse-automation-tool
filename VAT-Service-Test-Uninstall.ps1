#requires -Modules ScriptAsService
#requires -RunAsAdministrator

$BaseDir = "C:\VeVerse\veverse-automation-tool\"
$ServiceDir = "$BaseDir\Service"

Uninstall-ScriptAsService -Name "VeVerseAutomationTool_Test"
Remove-Item -Path "$ServiceDir" -Force