param(
    [string]$TemplatesDir = "$PSScriptRoot\..\internal\template\templates",
    [string]$OutFile = "$PSScriptRoot\..\templates_dump.txt"
)

$files = Get-ChildItem -Path $TemplatesDir -Recurse -Filter "*.tmpl" | Sort-Object FullName
$total = 0

$output = foreach ($f in $files) {
    $rel = $f.FullName.Substring((Resolve-Path $TemplatesDir).Path.Length + 1)
    $content = Get-Content $f.FullName -Raw
    $total += $content.Length
    "=== $rel ===`n$content`n"
}

$output | Out-File -FilePath $OutFile -Encoding utf8
Write-Host "Wrote $($files.Count) files ($total chars) to $OutFile"
