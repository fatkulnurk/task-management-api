$rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
$bytes = New-Object byte[] 48
$rng.GetBytes($bytes)
$rng.Dispose()

[Convert]::ToBase64String($bytes)
