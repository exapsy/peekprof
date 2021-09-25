#compdef peekprof

autoload -U is-at-least

_peekprof() {
  typeset -A opt_args
  typeset -a _arguments_options
  local ret=1

  if is-at-least 5.2; then
    _arguments_options=(-s -S -C)
  else
    _arguments_options=(-s -C)
  fi

  local context curcontext="$curcontext" state line
  _arguments "${_arguments_options[@]}" \
'-pid[process id to profile]: :(`ps -A -o pid | awk "NR > 1 { print }"`)' \
'-cmd[run and then profile the running command]:filename:' \
'-html[file output]:filename' \
'-csv[file output]:filename' \
'-refresh[refresh rate of profiling stats]:time' \
'-live[monitor process live]' \
'-livehost[host for the server which provides the live data]:' \
'-printoutput[show output of the command]' \
'-parent[monitor the parent and its children of the process provided by -pid]' \
&& ret=0
}

(( $+functions[_peekprof_commands] )) ||
_peekprof_commands() {
  local commands; commands=(

  )
  _describe -t commands 'peekprof commands' commands "$@"
}

peekprof "$@"
