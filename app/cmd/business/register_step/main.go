//go:build js && wasm

// register_step is the business-survival widget compiled as a
// WebAssembly module. It registers `stepSimulation` on the JS global
// and blocks forever so the Go runtime stays alive to service per-step
// calls from dexetera's runtime/worker.js.
//
// Build with the codegen-emitted app/business/build.sh or directly:
//
//	GOOS=js GOARCH=wasm go build -o app/business/src/main.wasm \
//	    ./app/cmd/business/register_step
package main

import (
	"github.com/umbralcalc/business-survival/app/pkg/businessdash"
	"github.com/umbralcalc/dexetera/pkg/simio"
)

func main() {
	simio.RegisterStep(businessdash.NewConfig())
}
