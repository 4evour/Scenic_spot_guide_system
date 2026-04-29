$headers = @{
    "Content-Type" = "application/json"
}

$body = '{"message":"井冈山在哪里"}'

$response = Invoke-WebRequest -Uri "http://localhost:9000/api/v1/ai/chat" -Method POST -Headers $headers -Body $body -UseBasicParsing
Write-Host "Status Code: $($response.StatusCode)"
Write-Host "Response: $($response.Content)"