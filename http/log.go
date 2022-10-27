package http

import (
	"fmt"
	"log"
	gohttp "net/http"
)

// LogWithRequest will prepend the remote address of 'req' to 'msg' before printing to 'logger'.
func LogWithRequest(logger *log.Logger, req *gohttp.Request, msg string, args ...interface{}) {
	msg = fmt.Sprintf("%s %s", req.RemoteAddr, msg)
	logger.Printf(msg, args...)
}
