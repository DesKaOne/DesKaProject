param(
 [Parameter(Mandatory=$true)][string]$Datadir,
 [Parameter(Mandatory=$true)][string]$Output,
 [ValidateSet("Zip","Directory")][string]$Format="Zip",
 [switch]$Help
)
if($Help){Write-Host "Usage: .\scripts\testnet-backup.ps1 -Datadir PATH -Output FILE [-Format Zip|Directory]";exit 0}
$ErrorActionPreference="Stop"
$source=(Resolve-Path $Datadir).Path
if(-not(Test-Path $source -PathType Container)){throw "datadir not found: $Datadir"}
if($Format -eq "Zip"){
 $target=$Output; if(-not $target.EndsWith(".zip")){$target="$target.zip"}
 $parent=Split-Path -Parent $target; if($parent){New-Item -ItemType Directory -Force -Path $parent|Out-Null}
 $tmp="$target.tmp.$([guid]::NewGuid().ToString('N'))"
 try{Compress-Archive -Path (Join-Path $source "*") -DestinationPath $tmp -CompressionLevel Optimal;Move-Item -Force $tmp $target}
 finally{if(Test-Path $tmp){Remove-Item -Force $tmp}}
}else{
 if(Test-Path $Output){throw "target already exists: $Output"}
 New-Item -ItemType Directory -Force -Path $Output|Out-Null
 Copy-Item -Path (Join-Path $source "*") -Destination $Output -Recurse -Force
}
Write-Host "backup created: $Output"
Write-Host "source datadir: $source"
