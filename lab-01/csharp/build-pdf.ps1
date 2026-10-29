# Renders PlantUML diagrams, generates report.tex (UTM template) from report.md, validates and compiles report.pdf.
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot
$java = 'D:\apps\temurin-jdk-21.0.12.1+1\bin\java.exe'
$plantuml = 'D:\apps\plantuml\plantuml.jar'
$pandoc = (Get-ChildItem D:\apps\pandoc -Recurse -Filter pandoc.exe | Select-Object -First 1).FullName

Push-Location assets
& $java -jar $plantuml -charset UTF-8 -tsvg diagrams.puml
& $java -jar $plantuml -charset UTF-8 -tpng -Sdpi=200 diagrams.puml
Pop-Location

python build_tex.py $pandoc
python D:\edu\faf\report-template\tools\check.py .\report.tex
D:\apps\tectonic\tectonic.exe report.tex
