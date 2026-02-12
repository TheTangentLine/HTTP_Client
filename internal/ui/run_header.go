package ui

import "fmt"

// PrintStepResult prints a preflight step result (e.g. DNS: OK) before the run header.
func PrintStepResult(name, value string, ok bool) {
	if ok {
		fmt.Printf("  %s%s%s : %s%s%s\n", colorDim, name, colorReset, colorGreen, value, colorReset)
	} else {
		fmt.Printf("  %s%s%s : %s%s%s\n", colorDim, name, colorReset, colorYellow, value, colorReset)
	}
}

// PrintRunHeader renders a colorful header for a single benchmark run.
func PrintRunHeader(url string, workers, connections, pipeline int, duration string) {
	fmt.Println()
	fmt.Printf("%s%sStarting HTTPCL benchmark%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf(" Target   : %s\n", url)
	fmt.Printf(" %s[workers:%s %s%d%s]  %s[connections:%s %s%d%s]  %s[pipeline:%s %s%d%s]  %s[duration:%s %s%s%s]\n",
		colorDim, colorReset, colorCyan, workers, colorReset,
		colorDim, colorReset, colorCyan, connections, colorReset,
		colorDim, colorReset, colorCyan, pipeline, colorReset,
		colorDim, colorReset, colorCyan, duration, colorReset,
	)
	fmt.Println()
}

