#!/usr/bin/env bash
#
###
rm -rf cvzf Go.tar.gz

tar cvzf Go.tar.gz ./*

scp Go.tar.gz zhangshifeng@10.16.49.129:/mnt/e/wsl/go
