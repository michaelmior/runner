package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/sethgrid/curse"
	"github.com/sethgrid/multibar"
	"github.com/smallfish/simpleyaml"
)

type Argument struct {
	key            string
	value          interface{}
	current, total int
}

// Produce the argument string for a flag with a given key and value
func flag_string(arg Argument) string {
	if _, ok := arg.value.(bool); ok {
		return fmt.Sprintf("--%s", arg.key)
	} else {
		return fmt.Sprintf("--%s=%v", arg.key, arg.value)
	}
}

func combinations(matrix [][]Argument) [][]Argument {
	return combinations_acc([]Argument{}, [][]Argument{}, matrix)
}

func combinations_acc(prefix []Argument, args [][]Argument, matrix [][]Argument) [][]Argument {
	if len(matrix) == 0 {
		return append(args, prefix)
	} else {
		new_args := append([][]Argument{}, args...)
		for _, margs := range matrix {
			for _, arg := range margs {
				new_prefixes := append([]Argument{arg}, prefix...)
				new_matrix := append(make([][]Argument, 0), matrix[1:]...)
				new_args = combinations_acc(new_prefixes, append(make([][]Argument, 0), new_args...), new_matrix)
			}
			break
		}

		return new_args
	}
}

func main() {
	// Load YAML config file
	data, err := ioutil.ReadFile("sample.yml")
	if err != nil {
		panic(err)
	}
	config, err := simpleyaml.NewYaml(data)
	if err != nil {
		panic(err)
	}

	cmd_vars, err := config.Get("vars").Array()
	if err != nil {
		panic(err)
	}

	var flags []Argument
	var params []string
	matrix := make([][]Argument, 0)

	for _, cmd_var := range cmd_vars {
		if cmd_str, ok := cmd_var.(string); ok {
			// This is a simple string parameter
			params = append(params, cmd_str)
		} else {
			// Anything not a string is assumed to be a map
			cmd_map := cmd_var.(map[interface{}]interface{})
			for key := range cmd_map {
				if cmd_val_arr, ok := cmd_map[key].([]interface{}); ok {
					// This is an array which we need to add to our run matrix
					var arg_arr []Argument
					for i, val := range cmd_val_arr {
						arg_arr = append(arg_arr, Argument{key.(string), val, i + 1, len(cmd_val_arr)})
					}
					matrix = append(matrix, arg_arr)
				} else {
					// This is a simple flag which we use for each iteration
					flags = append(flags, Argument{key.(string), cmd_map[key], 1, 1})
				}
			}
		}
	}

	// Generate all combinations of the matrix flags
	var matrix_flags [][]Argument
	if len(matrix) == 0 {
		matrix_flags = [][]Argument{make([]Argument, 0)}
	} else {
		matrix_flags = combinations(matrix)
	}

	run_script, err := config.GetPath("scripts", "run").String()
	if err != nil {
		panic(err)
	}

	// Ensure we can properly exit
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		<-sigc
		os.Exit(1)
	}()

	// Initialize progress bars
	bars, _ := multibar.New()
	progress := make([]func(int), len(matrix_flags[0]))
	for i := len(matrix_flags[0]) - 1; i >= 0; i-- {
		progress[i] = bars.MakeBar(matrix_flags[0][i].total, matrix_flags[0][i].key)
	}
	bars.Printf("\n") // print line to be overwritten by commands
	go bars.Listen()

	for _, mflags := range matrix_flags {
		// Combine all flags for this command
		cmd_flags := make([]string, 0)
		for _, flag := range append(flags, mflags...) {
			cmd_flags = append(cmd_flags, flag_string(flag))
		}
		cmd_flags = append(cmd_flags, params...)

		cmd := exec.Command(run_script, cmd_flags...)

		time.Sleep(250 * time.Millisecond)
		for i := len(mflags) - 1; i >= 0; i-- {
			progress[i](mflags[i].current)
		}
		fmt.Printf("\033[2K\r%s", cmd.Args)
		cmd.Output()

		// TODO: Execute the command and store the output
		// out, err := cmd.Output()
		// if err != nil {
		// 	panic(err)
		// }
		// fmt.Printf("%s", out)
	}

	// Ensure the terminal is correctly reset
	// XXX: for some reason, after resetting the terminal,
	//      we fail to return from main, so we just die after a brief wait
	go func() {
		<-time.After(100 * time.Millisecond)
		os.Exit(0)
	}()

	c, _ := curse.New()
	c.ModeRaw()
	c.ModeRestore()
}
