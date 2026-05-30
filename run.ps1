Get-Content .env | ForEach-Object {
    if ($_ -match '^\s*([^#=]+)=(.*)') {
        Set-Item -Path "env:$($matches[1].Trim())" -Value $matches[2].Trim()
    }
}
go run ./cmd/edge