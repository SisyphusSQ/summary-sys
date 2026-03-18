package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/SisyphusSQ/summary-sys/internal/collector"
	"github.com/SisyphusSQ/summary-sys/internal/formatter"
	"github.com/SisyphusSQ/summary-sys/internal/ssh"
	l "github.com/SisyphusSQ/summary-sys/pkg/log"
)

var (
	outputFormat string
	outputFile   string
	sshEnabled   bool
	sshHosts     string
	sshUser      string
	sshPort      int
	sshKeyPath   string
	sshPassword  string
	parallel     int
	timeout      int
)

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Collect and display system summary information",
	Long: `summary collects system information including CPU, memory, disk, 
network, processes, and load average. It supports both local and remote 
(SSH) collection.`,
	RunE: runSummary,
}

func init() {
	summaryCmd.Flags().StringVarP(&outputFormat, "format", "f", "text",
		"Output format: text, json")
	summaryCmd.Flags().StringVarP(&outputFile, "output", "o", "",
		"Output file (default: stdout)")

	summaryCmd.Flags().BoolVar(&sshEnabled, "ssh", false,
		"Collect from remote hosts via SSH")
	summaryCmd.Flags().StringVar(&sshHosts, "hosts", "",
		"SSH hosts (comma-separated, e.g., 192.168.1.1,192.168.1.2)")
	summaryCmd.Flags().StringVar(&sshUser, "ssh-user", "root",
		"SSH username")
	summaryCmd.Flags().IntVar(&sshPort, "ssh-port", 22,
		"SSH port")
	summaryCmd.Flags().StringVar(&sshKeyPath, "ssh-key", "",
		"SSH private key path")
	summaryCmd.Flags().StringVar(&sshPassword, "ssh-password", "",
		"SSH password")
	summaryCmd.Flags().IntVar(&parallel, "parallel", 5,
		"Number of parallel SSH connections")

	summaryCmd.Flags().IntVar(&timeout, "timeout", 30,
		"Collection timeout in seconds")

	rootCmd.AddCommand(summaryCmd)
}

type result struct {
	host   string
	output string
	err    error
}

func runSummary(cmd *cobra.Command, args []string) error {
	fmter, err := formatter.NewFormatter(outputFormat)
	if err != nil {
		return fmt.Errorf("create formatter: %w", err)
	}
	if fmter == nil {
		return fmt.Errorf("unknown output format: %s", outputFormat)
	}

	var results []string

	hostsList := strings.Split(sshHosts, ",")
	hostsList = func(list []string) []string {
		var ret []string
		for _, h := range list {
			h = strings.TrimSpace(h)
			if h != "" {
				ret = append(ret, h)
			}
		}
		return ret
	}(hostsList)

	if sshEnabled && len(hostsList) > 0 {
		numWorkers := parallel
		if numWorkers > len(hostsList) {
			numWorkers = len(hostsList)
		}

		jobs := make(chan string, len(hostsList))
		resultsCh := make(chan result, len(hostsList))

		var wg sync.WaitGroup
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for host := range jobs {
					r := collectRemoteHost(host, fmter, cmd)
					resultsCh <- r
				}
			}()
		}

		for _, host := range hostsList {
			jobs <- host
		}
		close(jobs)

		go func() {
			wg.Wait()
			close(resultsCh)
		}()

		for r := range resultsCh {
			if r.err != nil {
				l.Logger.Errorf("failed to collect from %s: %v", r.host, r.err)
				continue
			}
			results = append(results, r.output)
		}
	} else {
		l.Logger.Infof("collecting local system info")
		localCol := collector.NewLocalCollector()
		info, err := localCol.Collect(cmd.Context())
		if err != nil {
			return fmt.Errorf("collect local info: %w", err)
		}

		output, err := fmter.Format(info)
		if err != nil {
			return fmt.Errorf("format output: %w", err)
		}
		results = append(results, output)
	}

	for _, r := range results {
		if outputFile != "" {
			err := os.WriteFile(outputFile, []byte(r), 0644)
			if err != nil {
				return fmt.Errorf("write output file: %w", err)
			}
			l.Logger.Infof("output written to: %s", outputFile)
		} else {
			fmt.Println(r)
		}
	}

	return nil
}

func collectRemoteHost(host string, fmter formatter.Formatter, cmd *cobra.Command) result {
	l.Logger.Infof("collecting from remote host: %s", host)

	var auth ssh.AuthMethod
	if sshKeyPath != "" {
		auth = ssh.KeyAuth{KeyPath: sshKeyPath}
	} else if sshPassword != "" {
		auth = ssh.PasswordAuth{Password: sshPassword}
	} else {
		l.Logger.Infof("no SSH auth provided, trying SSH agent or default keys")

		agentAuth := ssh.AgentAuth{}
		defaultKeyAuth := ssh.DefaultKeyAuth{}

		if agentAuth.Available() {
			auth = agentAuth
		} else if defaultKeyAuth.Available() {
			auth = defaultKeyAuth
		} else {
			return result{host: host, err: fmt.Errorf("no SSH authentication available: SSH_AUTH_SOCK not set and no default keys found")}
		}
	}

	client, err := ssh.NewClient(&ssh.Config{
		Host:       host,
		Port:       sshPort,
		User:       sshUser,
		AuthMethod: auth,
	})
	if err != nil {
		return result{host: host, err: fmt.Errorf("failed to connect SSH: %w", err)}
	}
	defer client.Close()

	remoteCol, err := collector.NewRemoteCollector(client)
	if err != nil {
		return result{host: host, err: fmt.Errorf("failed to create remote collector: %w", err)}
	}

	info, err := remoteCol.Collect(cmd.Context())
	if err != nil {
		return result{host: host, err: fmt.Errorf("failed to collect: %w", err)}
	}

	output, err := fmter.Format(info)
	if err != nil {
		return result{host: host, err: fmt.Errorf("failed to format output: %w", err)}
	}

	return result{host: host, output: output}
}
