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
	Name         string
	Command      string
	OutputChecks []OutputCheck
	Commands     []Command
}

type OutputCheck struct {
	Pattern     string `yaml:"pattern"`
	CommandName string `yaml:"command_name"`
}

type Category struct {
	Name     string
	Commands []Command
}

type Config struct {
	Categories []Category `yaml:"categories"`
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
		fmt.Printf("Error reading configuration file: %v\n", err)
		os.Exit(1)
	}

	var commands []Command
	for _, category := range config.Categories {
		if category.Name == *osType {
			commands = category.Commands
			break
		}
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

	fmt.Printf("%s[%s] Executing %s...\n", strings.Repeat("  ", indent), color.YellowString("!"), command.Name)

	cmdStr := replaceVariables(command.Command, ip, domain)

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

	for _, outputCheck := range command.OutputChecks {
		if checkOutput(output, outputCheck.Pattern) {
			for _, subCommand := range command.Commands {
				if subCommand.Name == outputCheck.CommandName {
					wg.Add(1)
					go executeCommand(&subCommand, ip, domain, wg, indent+1)
					break
				}
			}
			break
		}
	}

	fmt.Printf("%s[%s] Completed %s. Result saved to %s\n", strings.Repeat("  ", indent), color.GreenString("✔"), command.Name, fileName)

	for i := range command.Commands {
		wg.Add(1)
		go executeCommand(&command.Commands[i], ip, domain, wg, indent+1)
	}
}

func checkOutput(output []byte, pattern string) bool {
	return strings.Contains(string(output), pattern)
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

func replaceVariables(command string, ip, domain string) string {
	command = strings.ReplaceAll(command, "{{.IP}}", ip)
	command = strings.ReplaceAll(command, "{{.Domain}}", domain)
	return command
}
