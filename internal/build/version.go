package build

// Version được gán lúc đóng gói; bản source không tự nhận là release đã cài.
var Version = development()

func development() string {
	const value = "dev"
	return value
}
