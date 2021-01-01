package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	_ "github.com/coredns/coredns/plugin/acl"
	_ "github.com/coredns/coredns/plugin/bind"
	_ "github.com/coredns/coredns/plugin/cache"
	_ "github.com/coredns/coredns/plugin/cancel"
	_ "github.com/coredns/coredns/plugin/errors"
	_ "github.com/coredns/coredns/plugin/forward"
	_ "github.com/coredns/coredns/plugin/health"
	_ "github.com/coredns/coredns/plugin/hosts"
	_ "github.com/coredns/coredns/plugin/log"
	_ "github.com/coredns/coredns/plugin/metrics"
	_ "github.com/coredns/coredns/plugin/template"
	_ "github.com/networkservicemesh/fanout"
)

var (
	versionID string
)

const (
	adBlockListURL = "https://github.com/notracking/hosts-blocklists/raw/master/hostnames.txt"
	adBlockFile    = "adservers.hosts"
)

var directives = []string{
	"acl",
	"bind",
	"cache",
	"cancel",
	"errors",
	"forward",
	"health",
	"hosts",
	"log",
	"prometheus",
	"template",
	"fanout",
}

func init() {
	caddy.Quiet = true
	caddy.AppName = "homedns"
	caddy.AppVersion = versionID
	dnsserver.Directives = directives
}

func main() {
	homedns := parseFlags()
	if homedns.printVersion {
		fmt.Fprintf(os.Stderr, "Version: %s", versionID)
		os.Exit(0)
	}
	input, err := homedns.corefile()
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}

	if homedns.dryRun {
		fmt.Fprintf(os.Stderr, string(input.Body()))
		os.Exit(0)
	}

	primedAdServers := make(chan bool)
	go func() {
		fmt.Fprintf(os.Stderr, "Starting adserver pull background job\n")
		ticker := time.NewTicker(4 * time.Hour)
		defer ticker.Stop()
		for {
			if err := retrieveAndWriteAdBlockHosts(adBlockListURL, adBlockFile); err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				continue
			}
			// First successful pull of the ad server block list file
			primedAdServers <- true
			// Run every 4 hours unless a failure occurs
			<-ticker.C
		}
	}()

	// Wait for first ad server pull
	<-primedAdServers
	fmt.Fprintf(os.Stderr, "AdServer BlockList Primed! Starting the DNS servers now!\n")

	instance, err := caddy.Start(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Blocking Known AdServers!\n")
	fmt.Fprintf(os.Stderr, "Sending Recursive Queries over TLS!\n")
	fmt.Fprintf(os.Stderr, "Sending to Multiple Recursive Services: \n")
	fmt.Fprintf(os.Stderr, "\t ✅ Google 8.8.8.8 8.8.4.4 \n")
	fmt.Fprintf(os.Stderr, "\t ✅ CloudFlare 1.1.1.1 1.0.0.1 \n")
	fmt.Fprintf(os.Stderr, "\t ✅ Quad9 9.9.9.9 149.112.112.112 \n")
	instance.Wait()
}

type homedns struct {
	printVersion bool
	dryRun       bool
	port         int
}

func parseFlags() *homedns {
	homedns := &homedns{}
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.BoolVar(&homedns.printVersion, "version", false, "Print Version")
	flags.BoolVar(&homedns.dryRun, "dry-run", false, "Print generated corefile and exit")
	flags.IntVar(&homedns.port, "port", 53, "UDP Port to listen on")
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "USAGE\n----\n%s [ options ]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOPTIONS\n----\n")
		flag.PrintDefaults()
	}
	flag.CommandLine = flags
	flag.Parse()
	return homedns
}

func (h *homedns) corefile() (caddy.Input, error) {
	var b bytes.Buffer
	corefileContents := fmt.Sprintf(corefileContentsTemplate, h.port, adBlockFile)
	_, err := b.WriteString(corefileContents)
	if err != nil {
		return nil, err
	}
	return caddy.CaddyfileInput{
		Contents:       b.Bytes(),
		Filepath:       "<flags>",
		ServerTypeName: "dns",
	}, nil
}

func retrieveAndWriteAdBlockHosts(adblockURL string, destFile string) error {
	httpClient := http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := httpClient.Get(adblockURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: Could not download latest adblock list from %s: %v\n", adblockURL, err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "WARNING: Received an HTTP %d status code when downloading adblock hosts from %s\n", resp.StatusCode, adblockURL)
		return fmt.Errorf("Could not download adblock hostsm got HTTP %d", resp.StatusCode)
	}
	hosts, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(destFile, hosts, 0644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Successfully wrote adserver block list to file\n")
	return nil
}

var corefileContentsTemplate = `.:%d {
	template ANY SOA local {
	   rcode NXDOMAIN
	}
	acl {
	   allow net 192.168.0.0/16 10.0.0.0/8 127.0.0.1/32 172.16.0.0/12 ::1
	   block
	}
	hosts %s {
	   fallthrough
	}
	fanout . 127.0.0.1:5301 127.0.0.1:5302 127.0.0.1:5303 {
	   except local
	}
	cancel
	errors
	log
	health
	cache 3600
	prometheus localhost:9253
 }
 
 .:5301 {
	bind 127.0.0.1
	forward . tls://8.8.8.8 tls://8.8.4.4 {
	   tls_servername dns.google
	   health_check 5s
	}
 }
 
 .:5302 {
	bind 127.0.0.1
	forward . tls://1.1.1.1 tls://1.0.0.1 {
	   tls_servername cloudflare-dns.com
	   health_check 5s
	}
 }
 
 .:5303 {
	bind 127.0.0.1
	forward . tls://9.9.9.9 tls://149.112.112.112 {
	   tls_servername dns.quad9.net
	   health_check 5s
	}
 }
`
