#!/bin/bash


docker push deciphernow/odrive  
docker push deciphernow/metadatadb 
docker push deciphernow/gatekeeper:latest 
docker push deciphernow/zk:latest 
docker push deciphernow/aac:latest 
docker push deciphernow/dias:latest

