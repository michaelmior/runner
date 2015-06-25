package main

import (
	"fmt"
	"io/ioutil"
	"os/exec"

	"github.com/smallfish/simpleyaml"
)

type Argument struct {
	key, value interface{}
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
					for _, val := range cmd_val_arr {
						arg_arr = append(arg_arr, Argument{key, val})
					}
					matrix = append(matrix, arg_arr)
				} else {
					// This is a simple flag which we use for each iteration
					flags = append(flags, Argument{key.(string), cmd_map[key]})
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

	for _, mflags := range matrix_flags {
		cmd_flags := make([]string, len(flags))
		for _, flag := range append(flags, mflags...) {
			cmd_flags = append(cmd_flags, flag_string(flag))
		}
		cmd_flags = append(cmd_flags, params...)

		cmd := exec.Command(run_script, cmd_flags...)
		fmt.Printf("%s\n", cmd.Args)

		// TOOD: Execute the command and store the output
		// out, err := cmd.Output()
		// if err != nil {
		// 	panic(err)
		// }
		// fmt.Printf("%s", out)
	}
}
