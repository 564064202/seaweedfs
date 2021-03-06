package shell

import (
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"

	"sort"

	"github.com/peterh/liner"
)

var (
	line        *liner.State
	historyPath = path.Join(os.TempDir(), "weed-shell")
)

func RunShell(options ShellOptions) {

	line = liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	setCompletionHandler()
	loadHistory()

	defer saveHistory()

	reg, _ := regexp.Compile(`'.*?'|".*?"|\S+`)

	commandEnv := NewCommandEnv(options)

	go commandEnv.MasterClient.KeepConnectedToMaster()
	commandEnv.MasterClient.WaitUntilConnected()

	for {
		cmd, err := line.Prompt("> ")
		if err != nil {
			if err != io.EOF {
				fmt.Printf("%v\n", err)
			}
			return
		}

		cmds := reg.FindAllString(cmd, -1)
		if len(cmds) == 0 {
			continue
		} else {
			line.AppendHistory(cmd)

			args := make([]string, len(cmds[1:]))

			for i := range args {
				args[i] = strings.Trim(string(cmds[1+i]), "\"'")
			}

			cmd := strings.ToLower(cmds[0])
			if cmd == "help" || cmd == "?" {
				printHelp(cmds)
			} else if cmd == "exit" || cmd == "quit" {
				return
			} else {
				foundCommand := false
				for _, c := range Commands {
					if c.Name() == cmd {
						if err := c.Do(args, commandEnv, os.Stdout); err != nil {
							fmt.Fprintf(os.Stderr, "error: %v\n", err)
						}
						foundCommand = true
					}
				}
				if !foundCommand {
					fmt.Fprintf(os.Stderr, "unknown command: %v\n", cmd)
				}
			}

		}
	}
}

func printGenericHelp() {
	msg :=
		`Type:	"help <command>" for help on <command>
`
	fmt.Print(msg)

	sort.Slice(Commands, func(i, j int) bool {
		return strings.Compare(Commands[i].Name(), Commands[j].Name()) < 0
	})
	for _, c := range Commands {
		helpTexts := strings.SplitN(c.Help(), "\n", 2)
		fmt.Printf("  %-30s\t# %s \n", c.Name(), helpTexts[0])
	}
}

func printHelp(cmds []string) {
	args := cmds[1:]
	if len(args) == 0 {
		printGenericHelp()
	} else if len(args) > 1 {
		fmt.Println()
	} else {
		cmd := strings.ToLower(args[0])

		sort.Slice(Commands, func(i, j int) bool {
			return strings.Compare(Commands[i].Name(), Commands[j].Name()) < 0
		})

		for _, c := range Commands {
			if c.Name() == cmd {
				fmt.Printf("  %s\t# %s\n", c.Name(), c.Help())
			}
		}
	}
}

func setCompletionHandler() {
	line.SetCompleter(func(line string) (c []string) {
		for _, i := range Commands {
			if strings.HasPrefix(i.Name(), strings.ToLower(line)) {
				c = append(c, i.Name())
			}
		}
		return
	})
}

func loadHistory() {
	if f, err := os.Open(historyPath); err == nil {
		line.ReadHistory(f)
		f.Close()
	}
}

func saveHistory() {
	if f, err := os.Create(historyPath); err != nil {
		fmt.Printf("Error writing history file: %v\n", err)
	} else {
		line.WriteHistory(f)
		f.Close()
	}
}
