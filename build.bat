go build -o veverse-automation-tool.exe -ldflags "-s -w" .
copy .\veverse-automation-tool.exe \\PC\X\Gitlab-Runner\builds\xxxxx\0\artheon\artheonui\Binaries\Deployment\
copy .\.credentials \\PC\X\Gitlab-Runner\builds\xxxxx\0\artheon\artheonui\Binaries\Deployment\
copy .\veverse-automation-tool.exe X:\UnrealEngine\Metaverse\Binaries\Deployment\
copy .\.credentials X:\UnrealEngine\Metaverse\Binaries\Deployment\