package main

import "os"

func privileged() bool                 { const root = 0; return os.Geteuid() == root }
func serviceMode(string) (bool, error) { return false, nil }
