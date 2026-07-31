param(
    [ValidateSet(
        "help",
        "build",
        "build-debug",
        "build-release",
        "clean",
        "test",
        "test-go",
        "test-release",
        "test-rust",
        "test-all",
        "bench",
        "bench-go",
        "bench-rust",
        "lint",
        "lint-go",
        "lint-rust",
        "lint-workflows",
        "fmt",
        "fmt-go",
        "fmt-rust"
    )]
    [string]$Task = "help"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$shimDir = Join-Path $repoRoot "shim"

function Get-TargetDirectory {
    $configured = $env:CARGO_TARGET_DIR
    if ([string]::IsNullOrWhiteSpace($configured)) {
        return Join-Path $shimDir "target"
    }

    if ([System.IO.Path]::IsPathRooted($configured)) {
        return $configured
    }

    return Join-Path $shimDir $configured
}

function Get-ShimLibraryPath {
    param(
        [Parameter(Mandatory = $true)]
        [ValidateSet("debug", "release")]
        [string]$Profile
    )

    $targetDir = Get-TargetDirectory
    $profileDir = Join-Path $targetDir $Profile
    return Join-Path $profileDir "chroma_shim.dll"
}

function Test-CommandAvailable {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$Hint
    )

    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Required command '$Name' not found. $Hint"
    }
}

function Invoke-CommandChecked {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter()][string[]]$Arguments = @(),
        [Parameter()][string]$WorkingDirectory = ""
    )

    $exitCode = $null
    $global:LASTEXITCODE = $null

    if ([string]::IsNullOrWhiteSpace($WorkingDirectory)) {
        & $Name @Arguments
        $exitCode = $global:LASTEXITCODE
    } else {
        Push-Location $WorkingDirectory
        try {
            & $Name @Arguments
            $exitCode = $global:LASTEXITCODE
        } finally {
            Pop-Location
        }
    }

    if ($null -eq $exitCode) {
        throw "Command '$Name' did not set LASTEXITCODE. Ensure it resolves to a native executable."
    }

    if ($exitCode -ne 0) {
        $argsText = $Arguments -join " "
        throw "Command failed with exit code ${exitCode}: $Name $argsText"
    }
}

function Invoke-WithChromaLibPath {
    param(
        [Parameter(Mandatory = $true)][string]$LibraryPath,
        [Parameter(Mandatory = $true)][scriptblock]$Action
    )

    if (-not (Test-Path $LibraryPath)) {
        throw "Expected library not found at '$LibraryPath'"
    }

    $resolved = (Resolve-Path $LibraryPath).Path
    $hadPrevious = Test-Path Env:CHROMA_LIB_PATH
    $previousValue = $env:CHROMA_LIB_PATH
    $env:CHROMA_LIB_PATH = $resolved

    try {
        & $Action
    } finally {
        if ($hadPrevious) {
            $env:CHROMA_LIB_PATH = $previousValue
        } else {
            if (Test-Path Env:CHROMA_LIB_PATH) {
                Remove-Item Env:CHROMA_LIB_PATH -ErrorAction SilentlyContinue
            }
        }
    }
}

function Build-Debug {
    Test-CommandAvailable -Name "cargo" -Hint "Install Rust with rustup (MSVC toolchain)."
    Invoke-CommandChecked -Name "cargo" -Arguments @("build", "--locked") -WorkingDirectory $shimDir

    $debugLib = Get-ShimLibraryPath -Profile "debug"
    if (-not (Test-Path $debugLib)) {
        throw "Debug build completed but '$debugLib' was not found. Check CARGO_TARGET_DIR and target configuration."
    }
    Write-Host "Built debug library at $debugLib"
}

function Build-Release {
    Test-CommandAvailable -Name "cargo" -Hint "Install Rust with rustup (MSVC toolchain)."
    Invoke-CommandChecked -Name "cargo" -Arguments @("build", "--locked", "--release") -WorkingDirectory $shimDir

    $releaseLib = Get-ShimLibraryPath -Profile "release"
    if (-not (Test-Path $releaseLib)) {
        throw "Release build completed but '$releaseLib' was not found. Check CARGO_TARGET_DIR and target configuration."
    }
    Write-Host "Built release library at $releaseLib"
}

function Test-GoDebug {
    Test-CommandAvailable -Name "go" -Hint "Install Go 1.21+ and ensure 'go' is on PATH."
    Build-Debug
    $debugLib = Get-ShimLibraryPath -Profile "debug"
    Invoke-WithChromaLibPath -LibraryPath $debugLib -Action {
        Invoke-CommandChecked -Name "go" -Arguments @("test", "-v", "./...") -WorkingDirectory $repoRoot
    }
}

function Test-GoRelease {
    Test-CommandAvailable -Name "go" -Hint "Install Go 1.21+ and ensure 'go' is on PATH."
    Build-Release
    $releaseLib = Get-ShimLibraryPath -Profile "release"
    Invoke-WithChromaLibPath -LibraryPath $releaseLib -Action {
        Invoke-CommandChecked -Name "go" -Arguments @("test", "-v", "./...") -WorkingDirectory $repoRoot
    }
}

function Test-Rust {
    Test-CommandAvailable -Name "cargo" -Hint "Install Rust with rustup (MSVC toolchain)."
    Invoke-CommandChecked -Name "cargo" -Arguments @("test", "--locked") -WorkingDirectory $shimDir
}

function Bench-Go {
    Test-CommandAvailable -Name "go" -Hint "Install Go 1.21+ and ensure 'go' is on PATH."
    Build-Debug
    $debugLib = Get-ShimLibraryPath -Profile "debug"
    Invoke-WithChromaLibPath -LibraryPath $debugLib -Action {
        Invoke-CommandChecked -Name "go" -Arguments @("test", "-run", "^$", "-bench", ".", "-benchmem", "./...") -WorkingDirectory $repoRoot
    }
}

function Bench-Rust {
    Test-CommandAvailable -Name "cargo" -Hint "Install Rust with rustup (MSVC toolchain)."
    Invoke-CommandChecked -Name "cargo" -Arguments @("bench", "--locked", "--bench", "ffi_bench") -WorkingDirectory $shimDir
}

function Invoke-Clean {
    Test-CommandAvailable -Name "cargo" -Hint "Install Rust with rustup (MSVC toolchain)."
    Invoke-CommandChecked -Name "cargo" -Arguments @("clean") -WorkingDirectory $shimDir

    $testDataDir = Join-Path $repoRoot "chroma_test_data"
    if (Test-Path $testDataDir) {
        Remove-Item -Path $testDataDir -Recurse -Force -ErrorAction Stop
    }
    Write-Host "Cleaned build artifacts."
}

function Lint-Go {
    Test-CommandAvailable -Name "golangci-lint" -Hint "Install golangci-lint and ensure it is on PATH."
    Invoke-CommandChecked -Name "golangci-lint" -Arguments @("run", "./...") -WorkingDirectory $repoRoot
}

function Lint-Rust {
    Test-CommandAvailable -Name "cargo" -Hint "Install Rust with rustup (MSVC toolchain)."
    Invoke-CommandChecked -Name "cargo" -Arguments @("clippy", "--locked", "--", "-D", "warnings") -WorkingDirectory $shimDir
}

function Lint-Workflows {
    Test-CommandAvailable -Name "go" -Hint "Install Go 1.21+ with automatic toolchain switching enabled, or install a local Go 1.24+ toolchain when switching is unavailable or disabled (for example, GOTOOLCHAIN=local or an older pinned GOTOOLCHAIN value), and ensure 'go' is on PATH."
    Test-CommandAvailable -Name "shellcheck" -Hint "Install ShellCheck from its official Windows release or with Chocolatey, then ensure 'shellcheck' is on PATH."
    Test-CommandAvailable -Name "yamllint" -Hint "Install yamllint with 'py -m pip install --user yamllint', then ensure 'yamllint' is on PATH."

    $actionlintVersionFile = Join-Path $repoRoot ".actionlint-version"
    if (-not (Test-Path -Path $actionlintVersionFile -PathType Leaf)) {
        throw ".actionlint-version is required for workflow linting but was not found at '$actionlintVersionFile'."
    }

    $actionlintVersion = (Get-Content -Path $actionlintVersionFile -Raw).Trim()
    if ([string]::IsNullOrWhiteSpace($actionlintVersion)) {
        throw ".actionlint-version must contain a non-empty actionlint version."
    }

    $shellcheckPath = (Get-Command "shellcheck" -ErrorAction Stop).Source
    if ([string]::IsNullOrWhiteSpace($shellcheckPath)) {
        throw "Unable to resolve the ShellCheck executable path."
    }

    $actionlintModule = "github.com/rhysd/actionlint/cmd/actionlint@$actionlintVersion"
    Write-Host "ShellCheck executable: $shellcheckPath"
    Invoke-CommandChecked -Name $shellcheckPath -Arguments @("--version")
    Write-Host "yamllint executable: $((Get-Command "yamllint" -ErrorAction Stop).Source)"
    Invoke-CommandChecked -Name "yamllint" -Arguments @("--version")
    Write-Host "actionlint module: $actionlintModule"
    Invoke-CommandChecked -Name "go" -Arguments @(
        "run",
        $actionlintModule,
        "-shellcheck=$shellcheckPath",
        "-ignore",
        "SC2129"
    ) -WorkingDirectory $repoRoot
    Invoke-CommandChecked -Name "yamllint" -Arguments @("-c", ".yamllint", ".") -WorkingDirectory $repoRoot
}

function Fmt-Go {
    Test-CommandAvailable -Name "gofmt" -Hint "Install Go 1.21+ and ensure gofmt is on PATH."
    Test-CommandAvailable -Name "goimports" -Hint "Install goimports: 'go install golang.org/x/tools/cmd/goimports@latest'."
    Invoke-CommandChecked -Name "gofmt" -Arguments @("-w", ".") -WorkingDirectory $repoRoot
    Invoke-CommandChecked -Name "goimports" -Arguments @("-w", ".") -WorkingDirectory $repoRoot
}

function Fmt-Rust {
    Test-CommandAvailable -Name "cargo" -Hint "Install Rust with rustup (MSVC toolchain)."
    Invoke-CommandChecked -Name "cargo" -Arguments @("fmt") -WorkingDirectory $shimDir
}

function Show-Help {
    Write-Host "Windows Developer Workflow for chroma-go-local"
    Write-Host ""
    Write-Host "Usage:"
    Write-Host "  pwsh -File .\scripts\dev-windows.ps1 -Task <task>"
    Write-Host ""
    Write-Host "Tasks:"
    Write-Host "  help          Show this help text"
    Write-Host "  build         Build Rust shim (debug)"
    Write-Host "  build-debug   Build Rust shim (debug)"
    Write-Host "  build-release Build Rust shim (release)"
    Write-Host "  clean         Clean build artifacts"
    Write-Host "  test          Build debug shim + run Go tests"
    Write-Host "  test-go       Build debug shim + run Go tests"
    Write-Host "  test-release  Build release shim + run Go tests"
    Write-Host "  test-rust     Run Rust tests"
    Write-Host "  test-all      Run Go and Rust tests"
    Write-Host "  bench         Run Go and Rust benchmarks"
    Write-Host "  bench-go      Build debug shim + run Go benchmarks"
    Write-Host "  bench-rust    Run Rust criterion benchmark"
    Write-Host "  lint          Run Go, Rust, Actions, embedded-shell, and YAML lint"
    Write-Host "  lint-go       Run golangci-lint"
    Write-Host "  lint-rust     Run cargo clippy"
    Write-Host "  lint-workflows Run actionlint, ShellCheck, and repository-wide yamllint"
    Write-Host "  fmt           Format Go and Rust code"
    Write-Host "  fmt-go        Format Go code (gofmt + goimports)"
    Write-Host "  fmt-rust      Format Rust code (cargo fmt)"
}

switch ($Task) {
    "help" { Show-Help }
    "build" { Build-Debug }
    "build-debug" { Build-Debug }
    "build-release" { Build-Release }
    "clean" { Invoke-Clean }
    "test" { Test-GoDebug }
    "test-go" { Test-GoDebug }
    "test-release" { Test-GoRelease }
    "test-rust" { Test-Rust }
    "test-all" { Test-GoDebug; Test-Rust }
    "bench" { Bench-Go; Bench-Rust }
    "bench-go" { Bench-Go }
    "bench-rust" { Bench-Rust }
    "lint" { Lint-Go; Lint-Rust; Lint-Workflows }
    "lint-go" { Lint-Go }
    "lint-rust" { Lint-Rust }
    "lint-workflows" { Lint-Workflows }
    "fmt" { Fmt-Go; Fmt-Rust }
    "fmt-go" { Fmt-Go }
    "fmt-rust" { Fmt-Rust }
}
