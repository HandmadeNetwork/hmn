#!/bin/bash
cd "$(dirname "$0")"
kill $(cat data/cinera.pid)
