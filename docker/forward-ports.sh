#!/bin/bash


DOCKER_VM_NAME="decipher-dev"
PORT=9093

if [[ $1 = "" ]]
then
  echo "Using virtualbox environment named ${DOCKER_VM_NAME}" 
fi

if [[ $2 = "" ]]
then
  echo "Opening port ${PORT}" 
fi


# Check for the Virtualbox command line utility
if ! type VBoxManage > /dev/null
then
  echo "Cannot find VBoxManage utility. Put it on your PATH or open vm ports manually."
  exit 1
fi


VBoxManage controlvm $DOCKER_VM_NAME natpf1 "tcp-port${PORT},tcp,,${PORT},,${PORT}";


