#!/usr/bin/env bash

function retry()
{
        local n=0
        local try=$1
        local cmd="${@: 2}"
        [[ $# -le 1 ]] && {
        echo "Usage $0 <retry_number> <Command>"; }

        until [[ $n -ge $try ]]
        do
                $cmd && break || {
                        echo "Command Fail.."
                        ((n++))
                        echo "retry $n ::"
                        if [[ $n -ge $try ]]; then
                          set -eu
                          $cmd
                        fi
                        sleep 30;
                        }

        done
}

retry $*
