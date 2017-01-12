#!/bin/bash
# Script to generate file containing commands to load country names into REDIS dictionary from GEONAMES database.
# Takes countryInfo.txt file from GEONAMES database as input.

if [ -z "$1" ]
  then
    echo "Please specify genomes country file as argument."
    exit
  else
    echo "Generating country names from geonames countryInfo.txt file ..."
    awk -F'\t' 'NF && !/^[:space:]*#/ {print(toupper($5)"\t"$2)}' $1 | awk -F'\t' '{gsub("[, \t\r\n]+|^THE[ ,]+|[ ,]+THE$|[ ,]THE[ ,]"," ",$1); print("SET \"CTRY-TO-TRI:"$1"\" "$2)}' > countryname-to-tri-geonames.data
    echo "done"
fi
