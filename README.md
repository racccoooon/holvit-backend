## static files setup

- build the vite frontend project (`npm run build`)
- check the file `generate.go`, adjust the paths to the vite artifacts to match your local setup
- run `go generate` 
- build with the build tag `embed_static`
- switch to production environment or set `Development.AuthFrontendUrl` to an empty string
- profit