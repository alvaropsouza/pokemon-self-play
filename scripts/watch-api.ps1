$ErrorActionPreference = "SilentlyContinue"
$proc = $null

function Restart-Server {
    if ($proc -and -not $proc.HasExited) {
        $proc.Kill()
        $null = $proc.WaitForExit(3000)
    }
    Write-Host "building..."
    & go build -o bin/play.exe ./cmd/play
    if ($LASTEXITCODE -ne 0) { Write-Host "build failed, waiting for fix..."; return }
    $script:proc = Start-Process -PassThru -NoNewWindow bin/play.exe
    Write-Host "server started (pid $($script:proc.Id))"
}

Restart-Server

$watcher = New-Object System.IO.FileSystemWatcher (Resolve-Path .)
$watcher.IncludeSubdirectories = $true
$watcher.Filter = "*.go"
$watcher.EnableRaisingEvents = $true
$watcher.NotifyFilter = [IO.NotifyFilters]::LastWrite

Write-Host "watching .go files..."
while ($true) {
    $change = $watcher.WaitForChanged(
        [IO.WatcherChangeTypes]::Changed -bor [IO.WatcherChangeTypes]::Created, 500)
    if (-not $change.TimedOut) {
        Write-Host "changed: $($change.Name)"
        Start-Sleep -Milliseconds 200  # debounce: editor pode salvar em rafaga
        Restart-Server
    }
}
