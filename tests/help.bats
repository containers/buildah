#!/usr/bin/env bats

load helpers

# run 'buildah help', parse the output looking for 'Available Commands';
# return that list.
function buildah_commands() {
    run_buildah help "$@" |\
        awk '/^Available Commands:/{ok=1;next}/^Flags:/{ok=0}ok { print $1 }' |\
        grep .
}

function check_help() {
    local count=0
    local -A found

    for cmd in $(buildah_commands "$@"); do
        # Human-readable buildah command string, with multiple spaces collapsed
        command_string="buildah $* $cmd"
        command_string=${command_string//  / }  # 'buildah  x' -> 'buildah x'

        # help command and --help flag have the same output
        run_buildah help "$@" $cmd
        local full_help=$output

        # The line immediately after 'Usage:' gives us a 1-line synopsis
        usage=$(echo "$output" | grep -A1 '^Usage:' | tail -1)
        [ -n "$usage" ] || die "$command_string: no Usage message found"
        expr "$usage" : "^  $command_string" > /dev/null || die "$command_string: Usage string doesn't match command"

        # If usage ends in '[command]', recurse into subcommands
        if expr "$usage" : '.*\[command\]$' >/dev/null; then
            found[subcommands]=1
            check_help "$@" $cmd
            continue
        fi

        # Cross-check: if usage includes '[flags]', there must be a
        # longer 'Flags:' section in the full --help output; vice-versa,
        # if 'Flags:' is in full output, usage line must have '[flags]'.
        if expr "$usage" : '.*\[flags' >/dev/null; then
            if ! expr "$full_help" : ".*Flags:" >/dev/null; then
                die "$command_string: Usage includes '[flags]' but has no 'Flags:' subsection"
            fi
        elif expr "$full_help" : ".*Flags:" >/dev/null; then
            die "$command_string: --help has 'Flags:' section but no '[flags]' in synopsis"
        fi

        count=$(expr $count + 1)

    done

    run_buildah "$@" --help
    full_usage=$output

    # Any command that takes subcommands, must show usage if called without one.
    run_buildah "$@"
    expect_output "$full_usage"

    # 'NoSuchCommand' subcommand shows usage unless the command is root 'buildah' command.
    if [ -n "$*" ]; then
        run_buildah "$@" NoSuchCommand
        expect_output "$full_usage"
    else
        run_buildah 125 "$@" NoSuchCommand
        expect_output --substring "unknown command"
    fi

    # This can happen if the output of --help changes, such as between
    # the old command parser and cobra.
    [ $count -gt 0 ] || \
        die "Internal error: no commands found in 'buildah help $@' list"

    # Sanity check: make sure the special loops above triggered at least once.
    # (We've had situations where a typo makes the conditional never run in podman)
    if [ -z "$*" ]; then
        # This loop is copied from podman test and redundant for buildah now.
        # But this is kept for future extension.
        for i in subcommands; do
            if [[ -z ${found[$i]} ]]; then
                die "Internal error: '$i' subtest did not trigger"
            fi
        done
    fi

    # This can happen if the output of --help changes, such as between
    # the old command parser and cobra.
    [ $count -gt 0 ] || \
        die "Internal error: no commands found in 'buildah help list"

}

@test "buildah help - basic tests" {
    check_help
}
