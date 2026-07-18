package util

import "os"

func GetUserAgent() string {
	return os.Getenv("USER_AGENT")
}
