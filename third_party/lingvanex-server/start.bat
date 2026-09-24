@echo off
rem Run the Lingvanex translation server standalone (the Go app can also manage
rem it via `lingvanex.manage: true` in the config).
cd /d "%~dp0"
py ./server.py
