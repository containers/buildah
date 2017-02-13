package seccomp

import (
	"fmt"
	"reflect"
	"strings"

	rspec "github.com/opencontainers/runtime-spec/specs-go"
)

// RemoveAction takes the argument string that was passed with the --remove flag,
// parses it, and updates the Seccomp config accordingly
func RemoveAction(arguments string, config *rspec.Seccomp) error {
	if config == nil {
		return fmt.Errorf("Cannot remove action from nil Seccomp pointer")
	}

	var syscallsToRemove []string
	if strings.Contains(arguments, ",") {
		syscallsToRemove = strings.Split(arguments, ",")
	} else {
		syscallsToRemove = append(syscallsToRemove, arguments)
	}

	for _, syscall := range syscallsToRemove {
		for counter, syscallStruct := range config.Syscalls {
			if syscallStruct.Name == syscall {
				config.Syscalls = append(config.Syscalls[:counter], config.Syscalls[counter+1:]...)
			}
		}
	}
	return nil
}

// RemoveAllSeccompRules removes all seccomp syscall rules
func RemoveAllSeccompRules(config *rspec.Seccomp) error {
	if config == nil {
		return fmt.Errorf("Cannot remove action from nil Seccomp pointer")
	}
	newSyscallSlice := []rspec.Syscall{}
	config.Syscalls = newSyscallSlice
	return nil
}

// RemoveAllMatchingRules will remove any syscall rules that match the specified action
func RemoveAllMatchingRules(config *rspec.Seccomp, action string) error {
	if config == nil {
		return fmt.Errorf("Cannot remove action from nil Seccomp pointer")
	}

	seccompAction, err := parseAction(action)
	if err != nil {
		return err
	}

	syscallsToRemove := []string{}
	for _, syscall := range config.Syscalls {
		if reflect.DeepEqual(syscall.Action, seccompAction) {
			syscallsToRemove = append(syscallsToRemove, syscall.Name)
		}
	}

	for i := range syscallsToRemove {
		RemoveAction(syscallsToRemove[i], config)
	}

	return nil
}
