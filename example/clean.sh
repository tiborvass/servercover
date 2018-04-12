#!/bin/sh

for i in server client example.sock cover.sock cover.out; do
	rm -f $(dirname "$0")/"$i"
done
