package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/fatih/color"
	"gopkg.in/yaml.v2"
)

type Command struct {
	Name        string
	Command     string
	OutputCheck []string
	Commands    []Command
}

type Config struct {
	Linux   []Command
	Windows []Command
}

var (
	ip     = flag.String("ip", "", "IP address")
	domain = flag.String("dc", "", "Domain")
	osType = flag.String("os", "", "Operating System (linux or windows)")
)

func main() {
	flag.Parse()
	if *ip == "" || *domain == "" || (*osType != "linux" && *osType != "windows") {
		fmt.Println("Usage: swissEnum -ip <ip> -dc <domain> -os <linux/windows>")
		os.Exit(1)
	}

	config, err := readConfig("commands.yaml")
	if err != nil {
		fmt.Printf("Error reading the configuration file: %v\n", err)
		os.Exit(1)
	}

	var commands []Command
	if *osType == "linux" {
		commands = config.Linux
	} else {
		commands = config.Windows
	}

	var wg sync.WaitGroup

	for i := range commands {
		wg.Add(1)
		go executeCommand(&commands[i], *ip, *domain, &wg, 0)
	}

	wg.Wait()
	fmt.Printf("[%s] Execution of all commands completed.\n", color.GreenString("!"))
}

func executeCommand(command *Command, ip, domain string, wg *sync.WaitGroup, indent int) {
	defer wg.Done()

	cmdStr := command.Command

	fmt.Printf("%s[%s] Executing %s...\n", strings.Repeat("  ", indent), color.YellowString("!"), command.Name)

	cmd := exec.Command("bash", "-c", cmdStr)
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("%s[%s] Error executing %s: %v\n", strings.Repeat("  ", indent), color.RedString("X"), command.Name, err)
		return
	}

	fileName := fmt.Sprintf("%s.txt", command.Name)
	err = ioutil.WriteFile(fileName, output, 0644)

	if err != nil {
		fmt.Printf("%s[%s] Error saving %s to %s: %v\n", strings.Repeat("  ", indent), color.RedString("X"), command.Name, fileName, err)
		return
	}

	// Check the output
	if checkOutput(output, command.OutputCheck) {
		fmt.Printf("%s[%s] Completed %s. Result saved in %s\n", strings.Repeat("  ", indent), color.GreenString("✔"), command.Name, fileName)
	} else {
		fmt.Printf("%s[%s] Check failed for %s. The command does not return the desired output.\n", strings.Repeat("  ", indent), color.RedString("X"), command.Name)
	}

	// Execute dependent commands
	for i := range command.Commands {
		wg.Add(1)
		go executeCommand(&command.Commands[i], ip, domain, wg, indent+1)
	}
}

func checkOutput(output []byte, expectedOutput []string) bool {
	for _, expected := range expectedOutput {
		if !strings.Contains(string(output), expected) {
			return false
		}
	}
	return true
}

func readConfig(filePath string) (*Config, error) {
	config := &Config{}

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
