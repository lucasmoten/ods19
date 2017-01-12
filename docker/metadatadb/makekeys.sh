#!/bin/sh

openssl enc -aes-256-ctr -k someuberpassword -P -md sha1 > tempkey

rm tempkey


