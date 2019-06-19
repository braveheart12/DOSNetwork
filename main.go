package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/DOSNetwork/core/dosnode"
	"github.com/DOSNetwork/core/onchain"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh/terminal"
)

// Caching running node's process id.
const pidFile string = "./dos/dosclient.pid"
const logFile string = "./dos/doslog"
const dosPath string = "./dos"

func savePID(pid int) {

	file, err := os.Create(pidFile)
	if err != nil {
		fmt.Printf("Unable to create pid file : %v\n", err)
		os.Exit(1)
	}

	defer file.Close()

	_, err = file.WriteString(strconv.Itoa(pid))
	if err != nil {
		fmt.Printf("Unable to create pid file : %v\n", err)
		os.Exit(1)
	}

	file.Sync() // flush to disk

}

func runDos(credentialPath, passphrase string) {

	// Make arrangement to remove PID file upon receiving the SIGTERM from kill command
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill, syscall.SIGTERM)

	go func() {
		//defer profile.Start().Stop()

		//defer os.Exit(0)
		signalType := <-ch
		signal.Stop(ch)

		fmt.Println("Received signal type : ", signalType)

		// remove PID file
		os.Remove(pidFile)

	}()

	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Println("runDos err ", err)
		return
	}
	if workingDir == "/" {
		workingDir = "."
	}
	fErr, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("runDos err ", err)
		return
	}
	syscall.Dup2(int(fErr.Fd()), 1) /* -- stdout */
	syscall.Dup2(int(fErr.Fd()), 2) /* -- stderr */

	dosclient, err := dosnode.NewDosNode(dosPath, passphrase)
	if err != nil {
		fmt.Println(" err", err)
		return
	}
	dosclient.Start()

}

func makeRequest(f string) ([]byte, error) {

	tServer := "http://localhost:8080/" + f

	req, err := http.NewRequest("GET", tServer, nil)
	if err != nil {
		fmt.Println("makeRequest err ", err)
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("makeRequest err ", err)
		return nil, err
	}
	defer resp.Body.Close()
	r, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("makeRequest err ", err)
		return nil, err
	}
	return r, err
}

func actionStart(c *cli.Context) error {
	// check if daemon already running.
	if _, err := os.Stat(pidFile); err == nil {
		fmt.Println("Already running or ${PWD}/dosclient.pid file exist.")
		os.Exit(1)
	}
	password := os.Getenv("PASSPHRASE")
	for password == "" {
		fmt.Print("Enter Password: ")
		bytePassword, _ := terminal.ReadPassword(0)
		password = strings.TrimSpace(string(bytePassword))
	}
	os.Setenv("PASSPHRASE", password)
	cmd := exec.Command(os.Args[0], "run")
	cmd.Stdout = os.Stdout
	cmd.Start()
	savePID(cmd.Process.Pid)
	return nil
}

func actionStop(c *cli.Context) error {
	_, err := os.Stat(pidFile)
	if err == nil {
		data, err := ioutil.ReadFile(pidFile)
		if err != nil {
			fmt.Println("Not running")
			return err
		}

		ProcessID, err := strconv.Atoi(string(data))
		if err != nil {
			fmt.Println("Unable to read and parse process id found in ", pidFile)
			return err
		}

		process, err := os.FindProcess(ProcessID)
		if err != nil {
			fmt.Printf("Unable to find process ID [%v] with error %v \n", ProcessID, err)
			return err
		}

		// remove PID file
		os.Remove(pidFile)
		fmt.Printf("Killing process ID [%v] now.\n", ProcessID)

		// kill process and exit immediately
		err = process.Kill()
		if err != nil {
			fmt.Printf("Unable to kill process ID [%v] with error %v \n", ProcessID, err)
			return err
		}
		fmt.Printf("Killed process ID [%v]\n", ProcessID)
		return nil
	}
	fmt.Println("Not running.")
	return err
}

func readPasswordFromTerm() (password string) {
	for password == "" {
		bytePassword, _ := terminal.ReadPassword(0)
		password = strings.TrimSpace(string(bytePassword))
	}
	return
}

func actionShowStatus(c *cli.Context) error {
	r, err := makeRequest("/")
	if err == nil {
		fmt.Println(string(r))
		return nil
	}
	return err
}

func actionGroupFormation(c *cli.Context) error {
	r, err := makeRequest("/signalGroupFormation")
	if err == nil {
		fmt.Println("trigger guardian functions : \n", string(r))
		return nil
	}
	return err
}

func actionGroupDissolve(c *cli.Context) error {
	r, err := makeRequest("/signalGroupDissolve")
	if err == nil {
		fmt.Println("trigger guardian functions : \n", string(r))
		return nil
	}
	return err
}

func actionBootstrap(c *cli.Context) error {
	c.String("cid")
	r, err := makeRequest("/signalBootstrap?cid=" + c.String("cid"))
	if err == nil {
		fmt.Println("trigger guardian functions : \n", string(r))
		return nil
	}
	return err
}

func actionRnadom(c *cli.Context) error {
	r, err := makeRequest("/signalRandom")
	if err == nil {
		fmt.Println("trigger guardian functions : \n", string(r))
		return nil
	}
	return err
}

func actionCreateWallet(c *cli.Context) error {

	fmt.Print("Enter Password: ")
	first := readPasswordFromTerm()

	fmt.Print("\nRe-enter Password: ")
	second := readPasswordFromTerm()

	if first != second {
		fmt.Print("\nUnmatched Password")
		return errors.New("Unmatched Password")
	}
	err := onchain.GenEthkey(dosPath, first)
	if err != nil {
		fmt.Println("\n", err)
	} else {
		fmt.Println("\nwallet keystore is in ", dosPath)
	}
	return nil
}

func actionWalletInfo(c *cli.Context) error {
	return nil
}

func actionWalletBalance(c *cli.Context) error {
	r, err := makeRequest("/balance")
	if err == nil {
		fmt.Println("show balance: ", string(r))
		return nil
	}
	return err
}

// main
func main() {
	if len(os.Args) > 1 && strings.ToLower(os.Args[1]) == "run" {
		password := os.Getenv("PASSPHRASE")
		for password == "" {
			fmt.Print("Enter Password: ")
			bytePassword, _ := terminal.ReadPassword(0)
			password = strings.TrimSpace(string(bytePassword))
		}

		runDos("", password)
	}

	app := cli.NewApp()
	app.Name = "client"
	app.Version = "beta"
	app.Usage = "CLI for dos client"

	app.Commands = []cli.Command{
		{
			Name:   "start",
			Usage:  "Start a dos client daemon",
			Action: actionStart,
		},
		{
			Name:   "stop",
			Usage:  "Stop a dos client daemon",
			Action: actionStop,
		},
		{
			Name:   "status",
			Usage:  "show dos client status",
			Action: actionShowStatus,
		},
		{
			Name:  "guardian",
			Usage: "Guardian fucntions",
			Subcommands: []cli.Command{
				{
					Name:   "groupFormation",
					Usage:  "signal proxy to generate a new group",
					Action: actionGroupFormation,
				},
				{
					Name:   "groupDissolve",
					Usage:  "signal proxy to dissolve expired groups",
					Action: actionGroupDissolve,
				},
				{
					Name:   "bootStrap",
					Usage:  "signal proxy to bootstrape",
					Action: actionBootstrap,
					Flags: []cli.Flag{
						cli.StringFlag{Name: "cid, bootstrap campaign id"},
					},
				},
				{
					Name:   "random",
					Usage:  "signal proxy to generate a random number",
					Action: actionRnadom,
				},
			},
		},
		{
			Name:  "wallet",
			Usage: "Manage Node Wallet",
			Subcommands: []cli.Command{
				{
					Name:   "create",
					Usage:  "show wallet status",
					Action: actionCreateWallet,
				},
				{
					Name:   "balance",
					Usage:  "show wallet balance",
					Action: actionWalletBalance,
				},
				{
					Name:   "info",
					Usage:  "show wallet info",
					Action: actionWalletInfo,
				},
			},
		},
	}

	app.Run(os.Args)
}
