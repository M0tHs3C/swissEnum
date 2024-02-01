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

type Config struct {
	Windows []Command
}

var (
	ip     = flag.String("ip", "", "Indirizzo IP")
	domain = flag.String("dc", "", "Dominio")
	osType = flag.String("os", "", "Sistema Operativo (linux o windows)")
)

func main() {
	flag.Parse()

	if *ip == "" || *domain == "" || (*osType != "linux" && *osType != "windows") {
		fmt.Println("Usage: go run main.go -ip <ip> -dc <domain> -os <linux/windows>")
		os.Exit(1)
	}

	config, err := readConfig("commands.yaml")
	if err != nil {
		fmt.Printf("Errore nella lettura del file di configurazione: %v\n", err)
		os.Exit(1)
	}

	var commands []Command
	if *osType == "windows" {
		commands = config.Windows
	} else {
		fmt.Println("Sistema operativo non supportato.")
		os.Exit(1)
	}

	var wg sync.WaitGroup

	for i := range commands {
		wg.Add(1)
		go executeCommand(&commands[i], *ip, *domain, &wg, 0)
	}

	wg.Wait()
	fmt.Printf("[%s] Completato l'esecuzione di tutti i comandi.\n", color.GreenString("!"))
}

func executeCommand(command *Command, ip, domain string, wg *sync.WaitGroup, indent int) {
	defer wg.Done()

	cmdStr := command.Command

	fmt.Printf("%s[%s] Esecuzione di %s...\n", strings.Repeat("  ", indent), color.YellowString("!"), command.Name)

	cmd := exec.Command("bash", "-c", cmdStr)
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("%s[%s] Errore nell'esecuzione di %s: %v\n", strings.Repeat("  ", indent), color.RedString("X"), command.Name, err)
		return
	}

	fileName := fmt.Sprintf("%s.txt", command.Name)
	err = ioutil.WriteFile(fileName, output, 0644)

	if err != nil {
		fmt.Printf("%s[%s] Errore nel salvataggio di %s in %s: %v\n", strings.Repeat("  ", indent), color.RedString("X"), command.Name, fileName, err)
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

	fmt.Printf("%s[%s] Completato %s. Risultato salvato in %s\n", strings.Repeat("  ", indent), color.GreenString("✔"), command.Name, fileName)

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
