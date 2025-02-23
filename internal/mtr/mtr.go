package mtr

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var (
	// Default paths
	defaultMTRPath = "/opt/homebrew/sbin/mtr"
	sudoPath       = "/usr/bin/sudo"
	
	// Get MTR path from environment or use default
	mtrPath = func() string {
		if path := os.Getenv("MTR_PATH"); path != "" {
			return path
		}
		return defaultMTRPath
	}()
)

const (
	// ANSI color codes
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
)

// Column widths for table formatting
var columnWidths = map[string]int{
	"hop":   3,    // Hop number
	"loss":  6,    // Loss%
	"snt":   3,    // Sent
	"last":  7,    // Last
	"avg":   7,    // Avg
	"best":  7,    // Best
	"worst": 7,    // Worst
	"stdev": 7,    // StDev
	"host":  40,   // Hostname
}

// Config represents the configuration for running MTR
type Config struct {
	Hostname string
	Count    int
	Report   bool
}

// Result represents the result of running MTR
type Result struct {
	Output string
	Error  error
}

// HopData represents the data for a single hop in the MTR output
type HopData struct {
	Hop      int
	Hostname string
	IP       string
	Loss     float64
	Sent     int
	Last     float64
	Avg      float64
	Best     float64
	Worst    float64
	StDev    float64
}

func formatHeader() string {
	return "\nMTR Report\n==========\n\n"
}

func formatHeaderExplanation() string {
	return `Column Explanation:
Loss%%    : Percentage of packets lost at this hop
Snt      : Number of packets sent
Last     : The latency of the last packet sent (ms)
Avg      : Average latency of all packets (ms)
Best     : The best (lowest) latency observed (ms)
Wrst     : The worst (highest) latency observed (ms)
StDev    : Standard deviation of latencies (ms)
Hostname : Hostname or IP address of the hop

Color Indicators:
Red     : High packet loss (≥20%)
Yellow  : High latency (≥100ms)

`
}

func formatHostInfo(hostname string) string {
	return fmt.Sprintf("Target Host: %s\n\n", hostname)
}

func colorizeOutput(hops []HopData) string {
	var table strings.Builder
	
	// Write header
	table.WriteString(fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s\n",
		columnWidths["hop"], "Hop",
		columnWidths["loss"], "Loss%",
		columnWidths["snt"], "Snt",
		columnWidths["last"], "Last",
		columnWidths["avg"], "Avg",
		columnWidths["best"], "Best",
		columnWidths["worst"], "Wrst",
		columnWidths["stdev"], "StDev",
		columnWidths["host"], "Host"))
	
	// Write separator
	totalWidth := 0
	for _, width := range columnWidths {
		totalWidth += width + 2 // +2 for spacing
	}
	table.WriteString(strings.Repeat("-", totalWidth) + "\n")
	
	// Write data rows
	for _, hop := range hops {
		// Color code for loss percentage
		lossColor := colorGreen
		if hop.Loss > 20 {
			lossColor = colorRed
		} else if hop.Loss > 5 {
			lossColor = colorYellow
		}
		
		// Format host string
		hostStr := hop.Hostname
		if hop.IP != "" && hop.Hostname != hop.IP && !strings.Contains(hop.Hostname, hop.IP) {
			hostStr = fmt.Sprintf("%s (%s)", hop.Hostname, hop.IP)
		}
		
		// Write the row with colors
		table.WriteString(fmt.Sprintf("%-*d  %s%-*.1f%s  %-*d  %-*.1f  %-*.1f  %-*.1f  %-*.1f  %-*.1f  %-*s\n",
			columnWidths["hop"], hop.Hop,
			lossColor, columnWidths["loss"], hop.Loss, colorReset,
			columnWidths["snt"], hop.Sent,
			columnWidths["last"], hop.Last,
			columnWidths["avg"], hop.Avg,
			columnWidths["best"], hop.Best,
			columnWidths["worst"], hop.Worst,
			columnWidths["stdev"], hop.StDev,
			columnWidths["host"], hostStr))
	}
	
	return table.String()
}

func generateSummary(hops []HopData) string {
	if len(hops) == 0 {
		return "\nNo route data available.\n"
	}
	
	var summary strings.Builder
	summary.WriteString("\nSummary:\n")
	summary.WriteString("--------\n")
	
	// Find worst performing hops
	var worstLoss, worstLatency HopData
	worstLoss = hops[0]
	worstLatency = hops[0]
	
	for _, hop := range hops {
		if hop.Loss > worstLoss.Loss {
			worstLoss = hop
		}
		if hop.Avg > worstLatency.Avg {
			worstLatency = hop
		}
	}
	
	// Report worst loss
	if worstLoss.Loss > 0 {
		summary.WriteString(fmt.Sprintf("Worst packet loss at hop %d (%s): %.1f%%\n",
			worstLoss.Hop, worstLoss.Hostname, worstLoss.Loss))
	} else {
		summary.WriteString("No packet loss detected\n")
	}
	
	// Report worst latency
	summary.WriteString(fmt.Sprintf("Highest average latency at hop %d (%s): %.1f ms\n",
		worstLatency.Hop, worstLatency.Hostname, worstLatency.Avg))
	
	// Calculate end-to-end metrics
	if len(hops) > 0 {
		lastHop := hops[len(hops)-1]
		summary.WriteString(fmt.Sprintf("\nEnd-to-end metrics for %s:\n", lastHop.Hostname))
		summary.WriteString(fmt.Sprintf("  Average: %.1f ms\n", lastHop.Avg))
		summary.WriteString(fmt.Sprintf("  Best: %.1f ms\n", lastHop.Best))
		summary.WriteString(fmt.Sprintf("  Worst: %.1f ms\n", lastHop.Worst))
		summary.WriteString(fmt.Sprintf("  Standard Deviation: %.1f ms\n", lastHop.StDev))
	}
	
	return summary.String()
}

func parseOutput(output string, count int) []HopData {
	lines := strings.Split(output, "\n")
	hopMap := make(map[string]*HopData)
	
	// Track sequence numbers to match p lines with their corresponding hop
	seqMap := make(map[string]string) // maps sequence -> hop number
	
	// Track received pings per hop
	receivedPings := make(map[string]int)
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		
		recordType := parts[0]
		hopNum := parts[1]
		
		// Convert hop number to 1-based index for display
		hopNumInt, _ := strconv.Atoi(hopNum)
		hopNumInt++ // Convert to 1-based
		hopNum = strconv.Itoa(hopNumInt)
		
		// Initialize hop if not exists
		if _, exists := hopMap[hopNum]; !exists {
			hopMap[hopNum] = &HopData{
				Hop:      hopNumInt,
				Hostname: "???",
				IP:      "",
				Loss:    100.0,
				Sent:    count, // Set sent to total attempts from config
				Last:    0.0,
				Avg:     0.0,
				Best:    math.MaxFloat64,
				Worst:   0.0,
				StDev:   0.0,
			}
		}
		
		hop := hopMap[hopNum]
		
		switch recordType {
		case "h": // IP address
			if len(parts) >= 3 {
				hop.IP = parts[2]
				if hop.Hostname == "???" { // Only use IP as hostname if we don't have a DNS name
					hop.Hostname = parts[2]
				}
			}
			
		case "d": // DNS name
			if len(parts) >= 3 {
				hostname := strings.Join(parts[2:], " ")
				hop.Hostname = hostname
			}
			
		case "x": // New sequence
			if len(parts) >= 3 {
				seqMap[parts[2]] = hopNum
			}
			
		case "p": // Ping result
			if len(parts) >= 4 {
				// Match sequence number to get correct hop
				seq := parts[3]
				if hopForSeq, exists := seqMap[seq]; exists {
					hop = hopMap[hopForSeq]
					receivedPings[hopNum]++
					
					// Convert usec to ms
					usec, err := strconv.ParseFloat(parts[2], 64)
					if err == nil {
						ms := usec / 1000.0
						hop.Last = ms
						
						// Update Best/Worst
						if ms < hop.Best {
							hop.Best = ms
						}
						if ms > hop.Worst {
							hop.Worst = ms
						}
						
						// Update Average
						received := float64(receivedPings[hopNum])
						hop.Avg = (hop.Avg*(received-1) + ms) / received
						
						// Update StDev if we have more than one sample
						if received > 1 {
							sumSq := 0.0
							for i := 0; i < int(received-1); i++ {
								sumSq += (hop.Last - hop.Avg) * (hop.Last - hop.Avg)
							}
							hop.StDev = math.Sqrt(sumSq / (received - 1))
						}
					}
				}
			}
		}
	}
	
	// Convert map to sorted slice
	var result []HopData
	maxHop := 0
	for _, hop := range hopMap {
		if hop.Hop > maxHop {
			maxHop = hop.Hop
		}
	}
	
	// Initialize Best to 0 for hops with no successful pings
	for hopNum, hop := range hopMap {
		if hop.Best == math.MaxFloat64 {
			hop.Best = 0
		}
		
		// Calculate loss percentage based on received pings
		received := float64(receivedPings[hopNum])
		if count > 0 {
			hop.Loss = 100.0 * (float64(count) - received) / float64(count)
		} else {
			hop.Loss = 100.0
		}
	}
	
	// Build sorted result, removing duplicate last hops
	var lastHop *HopData
	for i := 1; i <= maxHop; i++ {
		if hop, exists := hopMap[strconv.Itoa(i)]; exists {
			// Skip if this is a duplicate of the last hop (same IP/hostname) and not the first hop
			if lastHop != nil && i > 1 && 
				((hop.IP != "" && hop.IP == lastHop.IP) || 
				(hop.Hostname != "???" && hop.Hostname == lastHop.Hostname)) {
				continue
			}
			result = append(result, *hop)
			lastHop = hop
		}
	}
	
	return result
}

// Run executes the MTR command with the given configuration
func Run(ctx context.Context, cfg Config) (*Result, error) {
	args := []string{"-n", mtrPath} // -n flag for sudo to avoid reading from stdin
	
	if cfg.Report {
		args = append(args, "--raw") // Use raw format for better parsing
	} else {
		args = append(args, "-n") // Don't resolve names in live mode
	}
	
	if cfg.Count > 0 {
		args = append(args, "-c", fmt.Sprintf("%d", cfg.Count))
	}
	
	// Add hostname
	args = append(args, cfg.Hostname)

	cmd := exec.CommandContext(ctx, sudoPath, args...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		if strings.Contains(outputStr, "command not found") {
			return nil, fmt.Errorf("mtr command not found - please install mtr using 'brew install mtr'")
		}
		if strings.Contains(outputStr, "socket: Permission denied") {
			return nil, fmt.Errorf("permission denied - try running with sudo")
		}
		if outputStr != "" {
			return nil, fmt.Errorf("mtr error: %v, output: %s", err, outputStr)
		}
		return nil, fmt.Errorf("mtr error: %v", err)
	}

	// Parse the output
	hops := parseOutput(outputStr, cfg.Count)
	
	// If no hops were found, check the raw output for error messages
	if len(hops) == 0 {
		if strings.Contains(outputStr, "Failure to resolve") {
			return nil, fmt.Errorf("failed to resolve hostname: %s", cfg.Hostname)
		}
		if strings.Contains(outputStr, "socket: Permission denied") {
			return nil, fmt.Errorf("permission denied - try running with sudo")
		}
		if strings.Contains(outputStr, "command not found") {
			return nil, fmt.Errorf("mtr command not found - please install mtr using 'brew install mtr'")
		}
		return nil, fmt.Errorf("no route data available\nRaw output:\n%s", outputStr)
	}
	
	// Combine all output components
	finalOutput := formatHeader() +
		formatHeaderExplanation() +
		formatHostInfo(cfg.Hostname) +
		colorizeOutput(hops) +
		generateSummary(hops)
	
	return &Result{
		Output: finalOutput,
		Error:  nil,
	}, nil
}
