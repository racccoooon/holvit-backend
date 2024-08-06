package main

//go:generate go run holvit/build-tools/generate-static -p embed -i ../frontend/auth/dist -n AuthStatic -o server/embed/auth_static.go -f embed_static

//go:generate go run holvit/build-tools/generate-manifest -p generated -i ../frontend/auth/dist/.vite/manifest.json -n AuthManifest -o services/generated/frontend_manifest_auth.go
