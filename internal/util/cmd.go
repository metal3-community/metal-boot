package util

import "regexp"

func ParseCommandLine(command string) []string {
	r := regexp.MustCompile(`\"[^\"]+\"|\S+`)
	return r.FindAllString(command, -1)
}
