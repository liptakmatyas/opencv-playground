#!/bin/bash

if [[ $# -ne 2 ]]; then
    echo "Usage: $(basename $0) <input_file> <output_file>" >&2
    exit 64
fi

input_file=$1
output_file=$2

ffmpeg \
    -i ${input_file} \
    -map 0 -map -0:a -c copy \
    ${output_file}

