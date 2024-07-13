$name="g-trader"

echo "Building for Windows..."
$env:GOARCH="amd64"
$env:GOOS="windows"
go build -ldflags="-s -w" -trimpath=1 -o ./bin/${name}-win.exe .

echo "Building for Linux..."
$env:GOOS="linux"
go build -ldflags="-s -w" -trimpath=1 -o ./bin/${name}-linux .

foreach ($arch in @("arm64","amd64")) {
  echo "Building for Mac ($arch)..."
  $env:GOOS="darwin"
  $env:GOARCH="$arch"
  go build -ldflags="-s -w" -trimpath=1 -o ./bin/${name}_mac_${arch} .
}
echo "Creating universal binary for Mac..."
lipo -create -output "bin/${name}-mac" bin/${name}_mac_arm64 bin/${name}_mac_amd64
Remove-Item bin/* -Include "${name}_mac_*"
Copy-Item config.txt bin