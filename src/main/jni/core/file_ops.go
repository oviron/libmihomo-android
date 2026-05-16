//go:build android && cgo

package main

import "os"

// invokeAction already runs handleAction in its own goroutine, so file ops
// do not need an inner one.
func handleDelFile(path string, result ActionResult) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			result.success("")
			return
		}
		result.error(err.Error())
		return
	}
	if fileInfo.IsDir() {
		err = os.RemoveAll(path)
	} else {
		err = os.Remove(path)
	}
	if err != nil {
		result.error(err.Error())
		return
	}
	result.success("")
}
