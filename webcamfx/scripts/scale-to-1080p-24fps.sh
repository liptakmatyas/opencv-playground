#!/bin/bash

if [[ $# -ne 2 ]]; then
    echo "Usage: $(basename $0) <input_file> <output_file>" >&2
    exit 64
fi

input_file=$1
output_file=$2

SCALE='1920:1080'
FPS=24

ffmpeg \
    -i ${input_file} \
    -vf scale=${SCALE},fps=${FPS} \
    ${output_file}

