package modproxy

import (
	"fmt"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	functions.HTTP("ModProxy", modProxy)
}

// modProxy writes "Hello, World!" to the HTTP response.
func modProxy(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello, World!")
}
