#!/bin/bash
( find ./ -name '*.go' -print0 | xargs -0 cat ) | wc -l
