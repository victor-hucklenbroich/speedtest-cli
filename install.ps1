# Installer for speedtest-cli (https://github.com/victor-hucklenbroich/speedtest-cli).
# Works on Windows PowerShell 5.1 and PowerShell 7+. Downloads the release
# archive for this machine's architecture, verifies its checksum, installs
# speedtest.exe, and puts it on your PATH:
#
#   irm https://raw.githubusercontent.com/victor-hucklenbroich/speedtest-cli/main/install.ps1 | iex
#
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$Repo    = "victor-hucklenbroich/speedtest-cli"
$Binary  = "speedtest.exe"
$BaseUrl = "https://github.com/$Repo"

if ($PSVersionTable.PSVersion.Major -ge 6 -and -not $IsWindows) {
    throw "install.ps1 is for Windows; on macOS/Linux use install.sh instead"
}

[Net.ServicePointManager]::SecurityProtocol = `
    [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

# --- Detect architecture -----------------------------------------------------
$RawArch = $env:PROCESSOR_ARCHITEW6432
if (-not $RawArch) { $RawArch = $env:PROCESSOR_ARCHITECTURE }
switch ($RawArch) {
    "AMD64" { $Arch = "amd64" }
    "ARM64" { $Arch = "arm64" }
    default { throw "install.ps1: unsupported architecture $RawArch" }
}

# --- Resolve the version -----------------------------------------------------
if ($env:VERSION) {
    $Version = $env:VERSION
    if ($Version -notlike "v*") { $Version = "v$Version" } # accept 1.2.0 and v1.2.0
} else {
    $Resp = Invoke-WebRequest -Uri "$BaseUrl/releases/latest" -Method Head -UseBasicParsing
    if ($Resp.BaseResponse.PSObject.Properties["ResponseUri"]) {
        $FinalUrl = $Resp.BaseResponse.ResponseUri.AbsoluteUri
    } else {
        $FinalUrl = $Resp.BaseResponse.RequestMessage.RequestUri.AbsoluteUri
    }
    $Version = ($FinalUrl -split "/")[-1]
    if ($Version -notlike "v*") {
        throw "install.ps1: could not determine the latest release (no releases yet?)"
    }
}

# --- Download and verify -----------------------------------------------------
$Archive = "speedtest-cli_windows_$Arch.zip"
$Url     = "$BaseUrl/releases/download/$Version/$Archive"
$Tmp     = Join-Path ([IO.Path]::GetTempPath()) "speedtest-install-$([Guid]::NewGuid().ToString('N'))"
New-Item -ItemType Directory -Path $Tmp | Out-Null

try {
    Write-Host "Downloading speedtest $Version (windows/$Arch)..."
    Invoke-WebRequest -Uri $Url -OutFile (Join-Path $Tmp $Archive) -UseBasicParsing
    Invoke-WebRequest -Uri "$BaseUrl/releases/download/$Version/checksums.txt" `
        -OutFile (Join-Path $Tmp "checksums.txt") -UseBasicParsing

    $Expected = $null
    foreach ($Line in Get-Content (Join-Path $Tmp "checksums.txt")) {
        $Parts = $Line -split "\s+"
        if ($Parts.Length -ge 2 -and $Parts[1] -eq $Archive) { $Expected = $Parts[0]; break }
    }
    if (-not $Expected) { throw "install.ps1: no entry for $Archive in checksums.txt" }
    $Actual = (Get-FileHash -Algorithm SHA256 (Join-Path $Tmp $Archive)).Hash
    if ($Actual -ne $Expected) { # -ne is case-insensitive, matching the hex casing difference
        throw "install.ps1: checksum mismatch for $Archive (expected $Expected, got $Actual)"
    }

    Expand-Archive -Path (Join-Path $Tmp $Archive) -DestinationPath (Join-Path $Tmp "x")
    $Extracted = Join-Path (Join-Path $Tmp "x") $Binary
    if (-not (Test-Path $Extracted)) { throw "install.ps1: $Binary not found in $Archive" }

    # --- Install -------------------------------------------------------------
    if ($env:BIN_DIR) {
        $BinDir = $env:BIN_DIR
        $ManagePath = $false # you picked the directory, so you own its PATH entry
    } else {
        $BinDir = Join-Path $env:LOCALAPPDATA "Programs\speedtest-cli"
        $ManagePath = $true
    }
    New-Item -ItemType Directory -Path $BinDir -Force | Out-Null
    $Target = Join-Path $BinDir $Binary
    Copy-Item -Path $Extracted -Destination $Target -Force

    Write-Host "Installed $Target"
    & $Target --version
    if ($LASTEXITCODE -ne 0) { throw "install.ps1: $Target --version failed" }

    # --- PATH ----------------------------------------------------------------
    $OnPath = ($env:Path -split ";" | Where-Object { $_ -eq $BinDir }).Count -gt 0
    if ($ManagePath) {
        $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if (-not ($UserPath -split ";" | Where-Object { $_ -eq $BinDir })) {
            $NewPath = if ($UserPath) { $UserPath.TrimEnd(";") + ";" + $BinDir } else { $BinDir }
            [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
            Write-Host "Added $BinDir to your user PATH; already-open terminals need a restart to see it."
        }
        if (-not $OnPath) { $env:Path = "$env:Path;$BinDir" } # current session
    } elseif (-not $OnPath) {
        Write-Host ""
        Write-Host "warning: $BinDir is not on your PATH."
    }
} finally {
    Remove-Item -Recurse -Force -Path $Tmp -ErrorAction SilentlyContinue
}
