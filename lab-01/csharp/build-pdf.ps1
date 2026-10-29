# Renders PlantUML diagrams, then report.md -> report.pdf (pandoc + headless Chromium).
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot
$java = 'D:\apps\temurin-jdk-21.0.12.1+1\bin\java.exe'
$plantuml = 'D:\apps\plantuml\plantuml.jar'
$pandoc = (Get-ChildItem D:\apps\pandoc -Recurse -Filter pandoc.exe | Select-Object -First 1).FullName
$chrome = (Get-ChildItem D:\apps\playwright-browsers -Recurse -Filter chrome-headless-shell.exe | Select-Object -First 1).FullName

Push-Location assets
& $java -jar $plantuml -charset UTF-8 -tsvg diagrams.puml
& $java -jar $plantuml -charset UTF-8 -tpng -Sdpi=200 diagrams.puml
Pop-Location

$html = Join-Path $env:TEMP 'lab01-report.html'
& $pandoc report.md -f gfm-implicit_figures -t html5 -s --embed-resources --css assets/report.css `
    --syntax-highlighting=tango --metadata pagetitle="Lab 1: Creational Design Patterns" -o $html
& $chrome --headless --disable-gpu --no-pdf-header-footer --print-to-pdf="$PWD\report.pdf" ("file:///" + ($html -replace '\\', '/'))
