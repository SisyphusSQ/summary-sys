package cmd

import (
	"fmt"
	"os"
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
	sshHosts     []string
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
	summaryCmd.Flags().StringSliceVar(&sshHosts, "hosts", []string{},
		"SSH hosts (can be repeated)")
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

	if sshEnabled && len(sshHosts) > 0 {
		numWorkers := parallel
		if numWorkers > len(sshHosts) {
			numWorkers = len(sshHosts)
		}

		jobs := make(chan string, len(sshHosts))
		resultsCh := make(chan result, len(sshHosts))

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

		for _, host := range sshHosts {
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
		return result{host: host, err: fmt.Errorf("SSH auth required: --ssh-key or --ssh-password")}
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
