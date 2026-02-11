package ui

import "fmt"

// PrintIntroBanner renders a big ASCII "HTTPCL" banner similar in spirit to
// classic TUI splash screens. It uses only ASCII characters to respect the
// project's constraints.
func PrintIntroBanner() {
	logo := []string{
		"██╗  ██╗████████╗████████╗██████╗  ██████╗██╗     ",
		"██║  ██║╚══██╔══╝╚══██╔══╝██╔══██╗██╔════╝██║     ",
		"███████║   ██║      ██║   ██████╔╝██║     ██║     ",
		"██╔══██║   ██║      ██║   ██╔    ╗██║     ██║     ",
		"██║  ██║   ██║      ██║   ██║     ███████╗███████╗",
		"╚═╝  ╚═╝   ╚═╝      ╚═╝   ╚═╝     ╚═════╝╚══════╝",
	}

	fmt.Println()
	for _, line := range logo {
		fmt.Println(line)
	}
	fmt.Println()
	fmt.Println()
}
