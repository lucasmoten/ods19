#!/bin/bash
go-bindata migrations schema ../../defaultcerts/client-mysql/id ../../defaultcerts/client-mysql/trust
go build