package ui

import "fmt"

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

